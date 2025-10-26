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
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

type DoTaskOpts struct {
	DisableRegisterWorkflow bool
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
	Func TemporalWorkflowFunc
	Name string
}

func (t *DoTaskBuilder) Build() (TemporalWorkflowFunc, error) {
	tasks := make([]workflowFunc, 0)

	for _, task := range *t.task.Do {
		l := log.With().Str("task", task.Key).Logger()

		// Build task reference

		// Should this task be run?
		l.Debug().Msg("Checking if task can be run")

		// Check for a switch task
		l.Debug().Msg("Checking for switch task")

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
				Func: fn,
				Name: builder.GetTaskName(),
			})
		}
	}

	// Execute the workflow
	wf := t.workflowExecutor(tasks)

	if !t.opts.DisableRegisterWorkflow {
		log.Debug().Str("name", t.GetTaskName()).Msg("Registering workflow")
		t.temporalWorker.RegisterWorkflowWithOptions(wf, workflow.RegisterOptions{
			Name: t.GetTaskName(),
		})
	}

	return wf, nil
}

// workflowExecutor executes the workflow by iterating through the tasks in order
func (t *DoTaskBuilder) workflowExecutor(tasks []workflowFunc) TemporalWorkflowFunc {
	return func(ctx workflow.Context, input any, state *utils.State) (*utils.State, error) {
		logger := workflow.GetLogger(ctx)
		logger.Info("Running workflow", "workflow", t.GetTaskName())

		timeout := defaultWorkflowTimeout
		if t.doc.Timeout != nil && t.doc.Timeout.Timeout != nil && t.doc.Timeout.Timeout.After != nil {
			timeout = utils.ToDuration(t.doc.Timeout.Timeout.After)
		}
		logger.Debug("Setting activity options", "startToCloseTimeout", timeout)
		ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: timeout,
		})

		// Iterate through the tasks to create the workflow
		for _, task := range tasks {
			logger.Debug("Adding summary to activity context", "task", task.Name)
			ao := workflow.GetActivityOptions(ctx)
			ao.Summary = task.Name
			ctx = workflow.WithActivityOptions(ctx, ao)

			logger.Debug("Cloning state", "task", task.Name)
			state = state.Clone()

			logger.Info("Running task", "task", task.Name)
			_, err := task.Func(ctx, input, state)
			if err != nil {
				logger.Error("Error running task", "task", task.Name, "error", err)
				return nil, err
			}
		}

		return state, nil
	}
}
