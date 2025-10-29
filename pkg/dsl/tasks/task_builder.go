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
	"strings"

	"github.com/mrsimonemms/temporal-dsl/pkg/utils"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

var activities []any = make([]any, 0)

func ActivitiesList() []any {
	return activities
}

type TaskBuilder interface {
	Build() (TemporalWorkflowFunc, error)
	GetTask() model.Task
	GetTaskName() string
	ShouldRun(*utils.State) (bool, error)
}

type TemporalWorkflowFunc func(ctx workflow.Context, input any, state *utils.State) (output any, err error)

type builder[T model.Task] struct {
	doc            *model.Workflow
	name           string
	task           T
	temporalWorker worker.Worker
}

// This method is designed to be overridden
func (d *builder[T]) Build() (TemporalWorkflowFunc, error) {
	return nil, fmt.Errorf("task builder not implemented: %s", d.GetTaskName())
}

func (d *builder[T]) GetTask() model.Task {
	return d.task
}

func (d *builder[T]) GetTaskName() string {
	return d.name
}

func (d *builder[T]) ShouldRun(state *utils.State) (bool, error) {
	if ifStatement := d.task.GetBase().If; ifStatement != nil {
		res, err := utils.EvaluateString(ifStatement.String(), state)
		if err != nil {
			// Treat a parsing error as non-retryable
			return false, temporal.NewNonRetryableApplicationError("Error parsing if statement", "If statement error", err)
		}

		// Response can be a boolean, "TRUE" (case-insensitive) or "1"
		switch r := res.(type) {
		case bool:
			return r, nil
		case string:
			return strings.EqualFold(r, "TRUE") || r == "1", nil
		default:
			return false, temporal.NewNonRetryableApplicationError(
				"If statement response type unknown",
				"If statement error",
				fmt.Errorf("response not string or bool"),
			)
		}
	}

	// No statement - treat as true
	return true, nil
}

// Factory to create a TaskBuilder instance, or die trying
func NewTaskBuilder(taskName string, task model.Task, temporalWorker worker.Worker, doc *model.Workflow) (TaskBuilder, error) {
	switch t := task.(type) {
	case *model.CallHTTP:
		return NewCallHTTPTaskBuilder(temporalWorker, t, taskName)
	case *model.DoTask:
		return NewDoTaskBuilder(temporalWorker, t, taskName, doc)
	case *model.ForkTask:
		return NewForkTaskBuilder(temporalWorker, t, taskName, doc)
	case *model.SetTask:
		return NewSetTaskBuilder(temporalWorker, t, taskName)
	case *model.WaitTask:
		return NewWaitTaskBuilder(temporalWorker, t, taskName)
	default:
		return nil, fmt.Errorf("unsupported task type '%T' for task '%s'", t, taskName)
	}
}
