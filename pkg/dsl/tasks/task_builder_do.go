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

	"github.com/rs/zerolog/log"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

func NewDoTaskBuilder(temporalWorker worker.Worker, task *model.DoTask, taskName string) (*DoTaskBuilder, error) {
	return &DoTaskBuilder{
		builder: builder[*model.DoTask]{
			name:           taskName,
			task:           task,
			temporalWorker: temporalWorker,
		},
	}, nil
}

type DoTaskBuilder struct {
	builder[*model.DoTask]
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
		builder, err := NewTaskBuilder(task.Key, task.Task, t.temporalWorker)
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
	return t.workflowExecutor(tasks), nil
}

// workflowExecutor executes the workflow by iterating through the tasks in order
func (t *DoTaskBuilder) workflowExecutor(tasks []workflowFunc) TemporalWorkflowFunc {
	return func(ctx workflow.Context, input any, state map[string]any) (any, error) {
		logger := workflow.GetLogger(ctx)
		logger.Info("Running workflow", "workflow", t.GetTaskName())

		// Iterate through the tasks to create the workflow
		for _, task := range tasks {
			logger.Debug("Adding summary to activity context", "name", task.Name)
			ao := workflow.GetActivityOptions(ctx)
			ao.Summary = task.Name
			ctx = workflow.WithActivityOptions(ctx, ao)

			// @todo(sje): handle the output
			logger.Info("Running task", "name", task.Name)
			_, err := task.Func(ctx, input, state)
			if err != nil {
				logger.Error("Error running task", "name", task.Name, "error", err)
				return nil, err
			}
		}

		// @todo(sje): return the output
		return "hello world", nil
	}
}
