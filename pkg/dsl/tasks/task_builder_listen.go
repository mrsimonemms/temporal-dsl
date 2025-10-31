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

package tasks

import (
	"fmt"
	"slices"
	"time"

	"github.com/mrsimonemms/temporal-dsl/pkg/utils"
	swUtil "github.com/serverlessworkflow/sdk-go/v3/impl/utils"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

type ListenTaskType string

const (
	ListenTaskTypeQuery  ListenTaskType = "query"
	ListenTaskTypeSignal ListenTaskType = "signal"
	ListenTaskTypeUpdate ListenTaskType = "update"
)

func NewListenTaskBuilder(
	temporalWorker worker.Worker,
	task *model.ListenTask,
	taskName string,
) (*ListenTaskBuilder, error) {
	return &ListenTaskBuilder{
		builder: builder[*model.ListenTask]{
			name:           taskName,
			task:           task,
			temporalWorker: temporalWorker,
		},
	}, nil
}

type ListenTaskBuilder struct {
	builder[*model.ListenTask]
}

func (t *ListenTaskBuilder) Build() (TemporalWorkflowFunc, error) {
	events, isAll, err := t.listEvents()
	if err != nil {
		return nil, err
	}

	timeout := time.Minute
	if timeoutInterface, ok := t.task.Metadata["timeout"]; ok {
		if timeoutStr, ok := timeoutInterface.(string); !ok {
			return nil, fmt.Errorf("timeout must be a string")
		} else {
			if dur, err := time.ParseDuration(timeoutStr); err != nil {
				return nil, fmt.Errorf("error parsing timeout to duration: %w", err)
			} else {
				timeout = dur
			}
		}
	}

	return func(ctx workflow.Context, input any, state *utils.State) (any, error) {
		logger := workflow.GetLogger(ctx)
		logger.Debug("Registering listeners")

		areAllComplete := make([]bool, 0)
		areAnyComplete := false
		await := true

		fn := func(key int) func() {
			return func() {
				if isAll {
					areAllComplete[key] = true
				} else {
					areAnyComplete = true
				}
			}
		}

		for i, event := range events {
			if isAll {
				areAllComplete = append(areAllComplete, false)
			}

			switch ListenTaskType(event.With.Type) {
			case ListenTaskTypeQuery:
				// Non-blocking
				await = false
				if err := t.configureQuery(ctx, event, state); err != nil {
					return nil, fmt.Errorf("error setting signal: %w", err)
				}
			case ListenTaskTypeSignal:
				// Blocking
				t.configureSignal(ctx, event, state, fn(i))
			case ListenTaskTypeUpdate:
				// Blocking
				if err := t.configureUpdate(ctx, event, state, fn(i)); err != nil {
					return nil, fmt.Errorf("error setting signal: %w", err)
				}
			}
		}

		if await {
			if err := t.await(ctx, timeout, isAll, areAnyComplete, areAllComplete); err != nil {
				return nil, err
			}
		}

		return nil, nil
	}, nil
}

func (t *ListenTaskBuilder) await(
	ctx workflow.Context, timeout time.Duration, isAll, areAnyComplete bool, areAllComplete []bool,
) error {
	logger := workflow.GetLogger(ctx)

	logger.Debug("Wait for listener", "task", t.GetTaskName())
	ok, err := workflow.AwaitWithTimeout(ctx, timeout, func() bool {
		// Calculate if the task has finished
		if isAll {
			logger.Debug("Waiting for all listeners to complete", "status", areAllComplete)
			return utils.SlicesEqual(areAllComplete, true)
		} else {
			logger.Debug("Waiting for first listening to complete", "state", areAnyComplete)
			return areAnyComplete
		}
	})
	if err != nil {
		logger.Error("Error creating listening await", "error", err, "task", t.GetTaskName())
		return err
	}
	if !ok {
		logger.Warn("Await timeout", "task", t.GetTaskName())
		return fmt.Errorf("timeout")
	}

	return nil
}

func (t *ListenTaskBuilder) configureQuery(
	ctx workflow.Context, event *model.EventFilter, state *utils.State,
) error {
	logger := workflow.GetLogger(ctx)

	handler := func() (any, error) {
		logger.Debug("New query received", "event", event.With.ID)

		return t.processReply(ctx, event, state)
	}

	return workflow.SetQueryHandlerWithOptions(ctx, event.With.ID, handler, workflow.QueryHandlerOptions{})
}

func (t *ListenTaskBuilder) configureSignal(
	ctx workflow.Context, event *model.EventFilter, state *utils.State, onSuccess func(),
) {
	logger := workflow.GetLogger(ctx)
	logger.Debug("Creating signal", "signal", event.With.ID)

	var inputData any

	r := workflow.GetSignalChannel(ctx, event.With.ID)

	// Wrap in a coroutine to allow Await to handle the timeout
	workflow.Go(ctx, func(ctx workflow.Context) {
		logger.Debug("Listening for signal")
		_ = r.Receive(ctx, &inputData)

		state.AddData(map[string]any{
			t.GetTaskName(): inputData,
		})

		onSuccess()
	})
}

func (t *ListenTaskBuilder) configureUpdate(
	ctx workflow.Context, event *model.EventFilter, state *utils.State, onSuccess func(),
) error {
	logger := workflow.GetLogger(ctx)

	handler := func(ctx workflow.Context, data any) (any, error) {
		logger.Debug("New update received", "event", event.With.ID)

		// Store the received data
		state.AddData(map[string]any{
			event.With.ID: data,
		})

		res, err := t.processReply(ctx, event, state)

		onSuccess()

		return res, err
	}

	return workflow.SetUpdateHandlerWithOptions(
		ctx,
		event.With.ID,
		handler,
		workflow.UpdateHandlerOptions{
			Validator: func(ctx workflow.Context, _ any) error {
				return nil
			},
		})
}

func (t *ListenTaskBuilder) listEvents() (events []*model.EventFilter, isAll bool, err error) {
	listen := t.task.Listen
	if listen.To == nil {
		listen.To = &model.EventConsumptionStrategy{}
	}

	if len(listen.To.All) > 0 {
		isAll = true
		events = listen.To.All
	} else if len(listen.To.Any) > 0 {
		events = listen.To.Any
	} else if listen.To.One != nil {
		// Treat a "one" as an all
		isAll = true
		events = []*model.EventFilter{listen.To.One}
	} else {
		err = fmt.Errorf("no listen task configured: %s", t.GetTaskName())
		return events, isAll, err
	}

	if len(events) == 0 {
		err = fmt.Errorf("no events defined: %s", t.GetTaskName())
		return events, isAll, err
	}

	// @todo(sje): configure the "until" EventConsumptionUntil for "any" events

	for _, i := range events {
		err = t.validateEventFilter(i)
		if err != nil {
			return events, isAll, err
		}
	}

	return events, isAll, err
}

func (t *ListenTaskBuilder) processReply(ctx workflow.Context, event *model.EventFilter, state *utils.State) (any, error) {
	logger := workflow.GetLogger(ctx)

	// Deep clone the additional map so we get the uninterpolated template out each time
	additional := swUtil.DeepClone(event.With.Additional)

	if tpl, ok := additional["data"]; ok {
		templateKey := "template"

		obj, err := utils.TraverseAndEvaluateObj(
			model.NewObjectOrRuntimeExpr(map[string]any{
				// Put in a map as the template could be anything
				templateKey: tpl,
			}),
			state,
		)
		if err != nil {
			logger.Error("Error parsing data", "event", event.With.ID)
			return nil, err
		}

		// Return the data
		logger.Debug("Replied from event", "event", event.With.ID)

		return obj[templateKey], nil
	}
	return nil, nil
}

func (t *ListenTaskBuilder) validateEventFilter(event *model.EventFilter) error {
	if event.With.ID == "" {
		return fmt.Errorf("listen task id is not set")
	}
	if event.With.Type == "" {
		return fmt.Errorf("listen task type is not set")
	}

	validTaskTypes := []ListenTaskType{
		ListenTaskTypeQuery,
		ListenTaskTypeSignal,
		ListenTaskTypeUpdate,
	}

	if !slices.Contains(validTaskTypes, ListenTaskType(event.With.Type)) {
		return fmt.Errorf("listen task type is not known: %s", event.With.Type)
	}

	return nil
}
