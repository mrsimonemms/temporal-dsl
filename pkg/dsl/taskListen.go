/*
 * Copyright 2025 Simon Emms <simon@simonemms.com>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package dsl

import (
	"fmt"
	"slices"
	"time"

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type TaskListenResponse struct {
	Conditional   string `json:"conditional,omitempty"`
	EventComplete bool   `json:"eventComplete"`
	TaskComplete  bool   `json:"taskComplete"`
}

type ListenTaskType string

const (
	ListenTaskTypeQuery  ListenTaskType = "query"
	ListenTaskTypeSignal ListenTaskType = "signal"
	ListenTaskTypeUpdate ListenTaskType = "update"
)

func configureQueryListener(ctx workflow.Context, event *model.EventFilter, data *Variables) error {
	logger := workflow.GetLogger(ctx)

	handler := func() (any, error) {
		logger.Debug("Received query")

		if d, ok := event.With.Additional["data"]; ok {
			value, err := Interpolate(d, data)
			if err != nil {
				logger.Error("Error interpolating data", "error", err)
				return nil, err
			}

			// Convert the output
			if event.With.DataContentType == "application/json" {
				logger.Debug("Converting query to Golang type")

				// Convert YAML to Golang type
				var err error
				value, err = FromYAML(value)
				if err != nil {
					logger.Error("Cannot convert to Golang type - ensure query data is a string for interpolation", "error", err)
					return nil, fmt.Errorf("ensure query data is a string for interpolation: %w", err)
				}
			}

			return value, nil
		}

		// Return the parsed data
		return data, nil
	}

	return workflow.SetQueryHandlerWithOptions(ctx, event.With.ID, handler, workflow.QueryHandlerOptions{})
}

func configureSignalListener(ctx workflow.Context, event *model.EventFilter, _ *Variables) error {
	logger := workflow.GetLogger(ctx)
	logger.Debug("Creating signal", "signal", event.With.ID)

	r := workflow.GetSignalChannel(ctx, event.With.ID)

	// @todo(sje): allow data to be received via signal
	// @todo(sje): ignore if timeout is set to 0 or "0"
	if timeout, ok := event.With.Additional["timeout"]; ok {
		logger.Debug("Adding timeout to signal receiver", "timeout", timeout)
		t, err := time.ParseDuration(timeout.(string))
		if err != nil {
			logger.Error("Unable to parse duration: %w", err)
			return fmt.Errorf("unable to parse duration: %w", err)
		}

		received, _ := r.ReceiveWithTimeout(ctx, t, nil)
		if !received {
			logger.Error("Signal not received within timeout")
			return fmt.Errorf("signal not received within timeout")
		}
		return nil
	}

	logger.Debug("Listening for signal")
	_ = r.Receive(ctx, nil)

	return nil
}

func configureUpdateListener(ctx workflow.Context, event *model.EventFilter, data *Variables, onSuccess func()) error {
	logger := workflow.GetLogger(ctx)

	handler := func(ctx workflow.Context, args HTTPData) (*TaskListenResponse, error) {
		// This is designed to give some debug information to the developer
		resp := &TaskListenResponse{}

		if statement, ok := event.With.Additional["if"]; ok {
			// Parse a conditional - only accept the update if it resolves to "true"
			conditional := MustParseVariables(statement.(string), data)

			if conditional != "true" {
				logger.Debug(
					"Conditional event received and resolved to false",
					"event", event.With.ID,
					"conditional", conditional,
					"statement", statement,
				)
				resp.Conditional = conditional
				return resp, nil
			}
		}

		onSuccess()

		resp.EventComplete = true

		return resp, nil
	}

	return workflow.SetUpdateHandlerWithOptions(ctx, event.With.ID, handler,
		workflow.UpdateHandlerOptions{
			Validator: func(ctx workflow.Context, args HTTPData) error {
				data.AddData(args)

				if d, ok := event.With.Additional["if"]; ok {
					if s, ok := d.(string); !ok {
						return fmt.Errorf("if is not a string: %+v", d)
					} else {
						if _, err := ParseVariables(s, data); err != nil {
							logger.Error("cannot parse data", "error", err)
							return fmt.Errorf("cannot parse data: %w", err)
						}
					}
				}

				return nil
			},
		},
	)
}

func listenConfigure(task *model.ListenTask, key string) (events []*model.EventFilter, isAll bool, err error) {
	isAll = false
	events = make([]*model.EventFilter, 0)

	if len(task.Listen.To.All) > 0 {
		isAll = true
		for k, i := range task.Listen.To.All {
			if err = validateEventFilter(i); err != nil {
				err = fmt.Errorf("%w: %s.%d", err, key, k)
				return events, isAll, err
			}
			events = append(events, i)
		}
	} else if len(task.Listen.To.Any) > 0 {
		for k, i := range task.Listen.To.Any {
			if err = validateEventFilter(i); err != nil {
				err = fmt.Errorf("%w: %s.%d", err, key, k)
				return events, isAll, err
			}
			events = append(events, i)
		}
	} else if task.Listen.To.One != nil {
		if err = validateEventFilter(task.Listen.To.One); err != nil {
			err = fmt.Errorf("%w: %s", err, key)
			return events, isAll, err
		}
		events = append(events, task.Listen.To.One)
	} else if task.Listen.To.Until != nil {
		err = fmt.Errorf("%w: listen.to.until", ErrUnsupportedTask)
		return events, isAll, err
	} else {
		err = ErrUnsetListenIDTask
		return events, isAll, err
	}

	return events, isAll, err
}

func listenTaskImpl(task *model.ListenTask, key string) (TemporalWorkflowFunc, error) {
	events, isAll, err := listenConfigure(task, key)
	if err != nil {
		return nil, err
	}

	return func(ctx workflow.Context, data *Variables, output map[string]OutputType) (map[string]OutputType, error) {
		logger := workflow.GetLogger(ctx)
		logger.Debug("Registering listeners")

		isAllComplete := make([]bool, 0)
		isAnyComplete := false
		await := false

		for i, event := range events {
			if isAll {
				isAllComplete = append(isAllComplete, false)
			}

			switch ListenTaskType(event.With.Type) {
			case ListenTaskTypeQuery:
				if err := configureQueryListener(ctx, event, data); err != nil {
					logger.Error("Error setting query", "id", event.With.ID, "error", err)
					return nil, fmt.Errorf("error setting query: %w", err)
				}
			case ListenTaskTypeSignal:
				if err := configureSignalListener(ctx, event, data); err != nil {
					logger.Error("Error setting signal", "id", event.With.ID, "error", err)
					return nil, fmt.Errorf("error setting signal: %w", err)
				}
			case ListenTaskTypeUpdate:
				await = true
				if err := configureUpdateListener(ctx, event, data, func() {
					logger.Debug("Listen event received", "event", event.With.ID)
					if isAll {
						isAllComplete[i] = true
					} else {
						isAnyComplete = true
					}
				}); err != nil {
					logger.Error("Error setting update", "id", event.With.ID, "error", err)
					return nil, fmt.Errorf("error setting update: %w", err)
				}
			}
		}

		// @todo(sje): figure out a way of customising the timeout
		timeout := time.Hour

		if await {
			if err := waitForListener(ctx, timeout, isAll, isAnyComplete, isAllComplete); err != nil {
				return nil, err
			}
		}

		return nil, nil
	}, nil
}

func waitForListener(ctx workflow.Context, timeout time.Duration, isAll, isAnyComplete bool, isAllComplete []bool) error {
	logger := workflow.GetLogger(ctx)
	logger.Debug("Listening for updates", "timeout", timeout)

	if ok, err := workflow.AwaitWithTimeout(ctx, timeout, func() bool {
		// Calculate if the task if finished
		if isAll {
			logger.Debug("Waiting for listener(s) to complete", "complete", isAllComplete)
			return SlicesEqual(isAllComplete, true)
		} else {
			logger.Debug("Waiting for listener to complete", "complete", isAnyComplete)
			return isAnyComplete
		}
	}); err != nil {
		logger.Error("Error waiting", "error", err)
		return fmt.Errorf("error waiting: %w", err)
	} else if !ok {
		logger.Warn("Await timeout")
		return temporal.NewTimeoutError(*enums.TIMEOUT_TYPE_SCHEDULE_TO_START.Enum(), nil)
	}

	return nil
}

func validateEventFilter(event *model.EventFilter) error {
	if event.With.ID == "" {
		return ErrUnsetListenIDTask
	}
	if event.With.Type == "" {
		return ErrUnsetListenTypeTask
	}

	validTaskTypes := []ListenTaskType{
		ListenTaskTypeQuery,
		ListenTaskTypeSignal,
		ListenTaskTypeUpdate,
	}

	if !slices.Contains(validTaskTypes, ListenTaskType(event.With.Type)) {
		return ErrUnknownListenTypeTask
	}

	return nil
}
