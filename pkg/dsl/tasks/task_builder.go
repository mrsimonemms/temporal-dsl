/*
 * Copyright 2025 Temporal DSL authors <https://github.com/mrsimonemms/temporal-dsl/graphs/contributors>
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

	"github.com/mrsimonemms/temporal-dsl/pkg/dsl/metadata"
	"github.com/mrsimonemms/temporal-dsl/pkg/utils"
	"github.com/rs/zerolog/log"
	swUtils "github.com/serverlessworkflow/sdk-go/v3/impl/utils"
	"github.com/serverlessworkflow/sdk-go/v3/model"
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
	ParseMetadata(workflow.Context, *utils.State) error
	PostLoad() error
	ShouldRun(*utils.State) (bool, error)
}

type TemporalWorkflowFunc func(ctx workflow.Context, input any, state *utils.State) (output any, err error)

type builder[T model.Task] struct {
	doc            *model.Workflow
	name           string
	task           T
	temporalWorker worker.Worker
}

func (d *builder[T]) GetTask() model.Task {
	return d.task
}

func (d *builder[T]) GetTaskName() string {
	return d.name
}

func (d builder[T]) ParseMetadata(ctx workflow.Context, state *utils.State) error {
	logger := workflow.GetLogger(ctx)

	task := d.GetTask().GetBase()

	if len(task.Metadata) == 0 {
		// No metadata set - continue
		return nil
	}

	// Clone the metadata to avoid pollution
	mClone := swUtils.DeepClone(task.Metadata)

	parsed, err := utils.TraverseAndEvaluateObj(model.NewObjectOrRuntimeExpr(mClone), state)
	if err != nil {
		return fmt.Errorf("error interpolating metadata: %w", err)
	}

	if search, ok := parsed[metadata.MetadataSearchAttribute]; ok {
		logger.Debug("Parsing search attributes")
		if err := metadata.ParseSearchAttributes(ctx, search); err != nil {
			logger.Error("Error parsing search attributes", "attributes", search, "error", err)
			return fmt.Errorf("error parsing search attributes: %w", err)
		}
	}

	return nil
}

func (d *builder[T]) PostLoad() error {
	log.Trace().Str("task", d.GetTaskName()).Msg("Task has no post load hook")
	return nil
}

func (d *builder[T]) ShouldRun(state *utils.State) (bool, error) {
	return utils.CheckIfStatement(d.task.GetBase().If, state)
}

// Factory to create a TaskBuilder instance, or die trying
func NewTaskBuilder(taskName string, task model.Task, temporalWorker worker.Worker, doc *model.Workflow) (TaskBuilder, error) {
	switch t := task.(type) {
	case *model.CallHTTP:
		return NewCallHTTPTaskBuilder(temporalWorker, t, taskName, doc)
	case *model.CallFunction:
		if t.Call == "activity" {
			return NewCallActivityTaskBuilder(temporalWorker, t, taskName, doc)
		}
		return nil, fmt.Errorf("unsupported call type '%s' for task '%s'", t.Call, taskName)
	case *model.DoTask:
		return NewDoTaskBuilder(temporalWorker, t, taskName, doc)
	case *model.ForTask:
		return NewForTaskBuilder(temporalWorker, t, taskName, doc)
	case *model.ForkTask:
		return NewForkTaskBuilder(temporalWorker, t, taskName, doc)
	case *model.ListenTask:
		return NewListenTaskBuilder(temporalWorker, t, taskName, doc)
	case *model.RaiseTask:
		return NewRaiseTaskBuilder(temporalWorker, t, taskName, doc)
	case *model.RunTask:
		return NewRunTaskBuilder(temporalWorker, t, taskName, doc)
	case *model.SetTask:
		return NewSetTaskBuilder(temporalWorker, t, taskName, doc)
	case *model.SwitchTask:
		return NewSwitchTaskBuilder(temporalWorker, t, taskName, doc)
	case *model.TryTask:
		return NewTryTaskBuilder(temporalWorker, t, taskName, doc)
	case *model.WaitTask:
		return NewWaitTaskBuilder(temporalWorker, t, taskName, doc)
	default:
		return nil, fmt.Errorf("unsupported task type '%T' for task '%s'", t, taskName)
	}
}

// Ensure the tasks meets the TaskBuilder type
var (
	_ TaskBuilder = &CallActivityTaskBuilder{}
	_ TaskBuilder = &CallHTTPTaskBuilder{}
	_ TaskBuilder = &DoTaskBuilder{}
	_ TaskBuilder = &ForTaskBuilder{}
	_ TaskBuilder = &ForkTaskBuilder{}
	_ TaskBuilder = &ListenTaskBuilder{}
	_ TaskBuilder = &RaiseTaskBuilder{}
	_ TaskBuilder = &RunTaskBuilder{}
	_ TaskBuilder = &SetTaskBuilder{}
	_ TaskBuilder = &SwitchTaskBuilder{}
	_ TaskBuilder = &TryTaskBuilder{}
	_ TaskBuilder = &WaitTaskBuilder{}
)
