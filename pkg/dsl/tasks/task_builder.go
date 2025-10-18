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

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

type TaskBuilder interface {
	Build() (TemporalWorkflowFunc, error)
	GetTaskName() string
}

type TemporalWorkflowFunc func(ctx workflow.Context, input any, state map[string]any) (output any, err error)

type builder[T comparable] struct {
	name           string
	task           T
	temporalWorker worker.Worker
}

// This method is designed to be overridden
func (d *builder[T]) Build() (TemporalWorkflowFunc, error) {
	return nil, fmt.Errorf("task builder not implemented: %s", d.GetTaskName())
}

func (d *builder[T]) GetTaskName() string {
	return d.name
}

// Factory to create a TaskBuilder instance, or die trying
func NewTaskBuilder(taskName string, task model.Task, temporalWorker worker.Worker) (TaskBuilder, error) {
	switch t := task.(type) {
	case *model.CallHTTP:
		return NewCallHTTPTaskBuilder(temporalWorker, t, taskName)
	case *model.DoTask:
		return NewDoTaskBuilder(temporalWorker, t, taskName)
	case *model.ForkTask:
		return NewForkTaskBuilder(temporalWorker, t, taskName)
	case *model.SetTask:
		return NewSetTaskBuilder(temporalWorker, t, taskName)
	case *model.WaitTask:
		return NewWaitTaskBuilder(temporalWorker, t, taskName)
	default:
		return nil, fmt.Errorf("unsupported task type '%T' for task '%s'", t, taskName)
	}
}
