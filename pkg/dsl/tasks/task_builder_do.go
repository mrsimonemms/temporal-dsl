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

	"github.com/mrsimonemms/temporal-dsl/pkg/utils"
	"github.com/rs/zerolog/log"
	swUtil "github.com/serverlessworkflow/sdk-go/v3/impl/utils"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

type DoTaskOpts struct {
	DisableRegisterWorkflow bool
	Envvars                 map[string]any
	Validator               *utils.Validator
}

func NewDoTaskBuilder(
	temporalWorker worker.Worker,
	task *model.DoTask,
	taskName string,
	doc *model.Workflow,
	opts ...DoTaskOpts,
) (*DoTaskBuilder, error) {
	var doOpts DoTaskOpts
	if len(opts) == 1 {
		doOpts = opts[0]
	}
	if doOpts.Envvars == nil {
		doOpts.Envvars = map[string]any{}
	}

	return &DoTaskBuilder{
		builder: builder[*model.DoTask]{
			doc:            doc,
			name:           taskName,
			task:           task,
			temporalWorker: temporalWorker,
		},
		opts: doOpts,
	}, nil
}

type DoTaskBuilder struct {
	builder[*model.DoTask]
	opts DoTaskOpts
}

type workflowFunc struct {
	TaskBuilder

	Func TemporalWorkflowFunc
	Name string
}

func (t *DoTaskBuilder) Build() (TemporalWorkflowFunc, error) {
	tasks := make([]workflowFunc, 0)

	var hasNoDo bool
	for _, task := range *t.task.Do {
		l := log.With().Str("task", task.Key).Logger()

		if do := task.AsDoTask(); do == nil {
			l.Debug().Msg("No do task detected")
			hasNoDo = true
		}

		// Build a task builder
		l.Debug().Msg("Creating task builder")
		builder, err := NewTaskBuilder(task.Key, task.Task, t.temporalWorker, t.doc)
		if err != nil {
			return nil, fmt.Errorf("error creating task builder: %w", err)
		}

		// Build the task and store it for use
		l.Debug().Msg("Building task")
		fn, err := builder.Build()
		if err != nil {
			return nil, fmt.Errorf("error building task: %w", err)
		}
		if fn != nil {
			tasks = append(tasks, workflowFunc{
				Func:        fn,
				Name:        builder.GetTaskName(),
				TaskBuilder: builder,
			})
		}
	}

	// Execute the workflow
	wf := t.workflowExecutor(tasks)

	if !t.opts.DisableRegisterWorkflow {
		if hasNoDo {
			log.Debug().Str("name", t.GetTaskName()).Msg("Registering workflow")
			t.temporalWorker.RegisterWorkflowWithOptions(wf, workflow.RegisterOptions{
				Name: t.GetTaskName(),
			})
		}
	}

	return wf, nil
}

// validateInput validates the input if it exists
func (t *DoTaskBuilder) validateInput(ctx workflow.Context, inputDef *model.Input, state *utils.State) error {
	logger := workflow.GetLogger(ctx)

	if inputDef != nil {
		logger.Debug("Validating input against schema")
		if err := swUtil.ValidateSchema(state.Input, inputDef.Schema, t.GetTaskName()); err != nil {
			logger.Error("Input failed data validation", "error", err)

			return temporal.NewNonRetryableApplicationError(
				"Workflow input did not meet JSON schema specification",
				"Validation",
				err,
				// There is additional detail useful in here
				err.(*model.Error),
			)
		}
	}

	return nil
}

// workflowExecutor executes the workflow by iterating through the tasks in order
func (t *DoTaskBuilder) workflowExecutor(tasks []workflowFunc) TemporalWorkflowFunc {
	return func(ctx workflow.Context, input any, state *utils.State) (any, error) {
		logger := workflow.GetLogger(ctx)
		logger.Info("Running workflow", "workflow", t.GetTaskName())

		if state == nil {
			logger.Debug("Creating new state instance")
			state = utils.NewState()
			state.Env = t.opts.Envvars
			state.Input = input

			// Validate input for the whole document
			logger.Debug("Validating input against document")
			if err := t.validateInput(ctx, t.doc.Input, state); err != nil {
				logger.Debug("Document input validation error", "error", err)
				return nil, err
			}
		}

		timeout := defaultWorkflowTimeout
		if t.doc.Timeout != nil && t.doc.Timeout.Timeout != nil && t.doc.Timeout.Timeout.After != nil {
			timeout = utils.ToDuration(t.doc.Timeout.Timeout.After)
		}
		logger.Debug("Setting activity options", "startToCloseTimeout", timeout)
		ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: timeout,
		})

		// Iterate through the tasks to create the workflow
		if err := t.iterateTasks(ctx, tasks, input, state); err != nil {
			return nil, err
		}

		return state.Output, nil
	}
}

func (t *DoTaskBuilder) iterateTasks(
	ctx workflow.Context, tasks []workflowFunc, input any, state *utils.State,
) error {
	var nextTargetName *string
	logger := workflow.GetLogger(ctx)

	for _, task := range tasks {
		taskBase := task.GetTask().GetBase()

		state.AddData(map[string]any{
			"task": map[string]any{
				"name": task.GetTaskName(),
			},
		})

		if nextTargetName != nil {
			logger.Debug("Check if a next task is set and it's this one", "task", task.Name)
			if task.Name == *nextTargetName {
				logger.Debug("Task is next one to be run from flow directive", "task", task.Name)
				// We've found the desired task - reset
				nextTargetName = nil
			} else {
				// Not the task - skip
				logger.Debug("Skipping task as not one set as next target", "task", task.Name, "nextTask", *nextTargetName)
				continue
			}
		}

		logger.Debug("Check if task should be run", "task", task.Name)
		if toRun, err := task.ShouldRun(state); err != nil {
			logger.Error("Error checking if statement", "error", err, "name", task.Name)
			return err
		} else if !toRun {
			logger.Debug("Skipping task as if statement resolve as false", "name", task.Name)
			continue
		}

		// Check input for the task
		logger.Debug("Validating input against task", "name", task.Name)
		if err := t.validateInput(ctx, taskBase.Input, state); err != nil {
			logger.Debug("Task input validation error", "error", err)
			return err
		}

		logger.Debug("Parse metadata", "name", task.Name)
		if err := task.ParseMetadata(ctx, state); err != nil {
			logger.Error("Error parsing metadata", "error", err)
			return err
		}

		logger.Debug("Adding summary to activity context", "name", task.Name)
		ao := workflow.GetActivityOptions(ctx)
		ao.Summary = task.Name
		ctx = workflow.WithActivityOptions(ctx, ao)

		logger.Info("Running task", "name", task.Name)
		output, err := task.Func(ctx, input, state)
		if err != nil {
			logger.Error("Error running task", "name", task.Name, "error", err)
			return err
		}

		// Set the output - this is only set if there's an export.as on the task
		state.AddOutput(task.GetTask(), output)

		if then := taskBase.Then; then != nil {
			flowDirective := then.Value
			if then.IsTermination() {
				logger.Debug("Workflow to be terminated", "flow", flowDirective)
				break
			}
			if !then.IsEnum() {
				logger.Debug("Next task targeted", "nextTask", flowDirective)
				nextTargetName = &flowDirective
				continue
			}
		}
	}

	if nextTargetName != nil {
		logger.Error("Next target specified but not found", "targetTask", nextTargetName)
		return fmt.Errorf("next target specified but not found: %s", *nextTargetName)
	}

	return nil
}
