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

	"github.com/mrsimonemms/temporal-dsl/pkg/utils"
	"github.com/rs/zerolog/log"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

func NewTryTaskBuilder(
	temporalWorker worker.Worker,
	task *model.TryTask,
	taskName string,
	doc *model.Workflow,
) (*TryTaskBuilder, error) {
	return &TryTaskBuilder{
		builder: builder[*model.TryTask]{
			doc:            doc,
			name:           taskName,
			task:           task,
			temporalWorker: temporalWorker,
		},
	}, nil
}

type TryTaskBuilder struct {
	builder[*model.TryTask]

	tryChildWorkflowName   string
	catchChildWorkflowName string
}

func (t *TryTaskBuilder) Build() (TemporalWorkflowFunc, error) {
	tasks := map[string]*model.TaskList{
		"try":   t.task.Try,
		"catch": t.task.Catch.Do,
	}

	for taskType, list := range tasks {
		name, err := t.registerTaskList(taskType, list)
		if err != nil {
			return nil, fmt.Errorf("erroring registring %s tasks for %s: %w", taskType, t.GetTaskName(), err)
		}

		if taskType == "try" {
			t.tryChildWorkflowName = name
		} else {
			t.catchChildWorkflowName = name
		}
	}

	return t.exec()
}

func (t *TryTaskBuilder) exec() (TemporalWorkflowFunc, error) {
	return func(ctx workflow.Context, input any, state *utils.State) (output any, err error) {
		logger := workflow.GetLogger(ctx)

		opts := workflow.ChildWorkflowOptions{
			WorkflowID: fmt.Sprintf("%s_try", workflow.GetInfo(ctx).WorkflowExecution.ID),
		}
		childCtx := workflow.WithChildOptions(ctx, opts)

		var res map[string]any
		if err := workflow.ExecuteChildWorkflow(childCtx, t.tryChildWorkflowName, state.Input, state).Get(ctx, &res); err != nil {
			logger.Warn("Workflow failed, catching the error", "tryWorkflow", t.tryChildWorkflowName, "catchWorkflow", t.catchChildWorkflowName)
			// The try workflow has failed - let's run the catch workflow
			opts := workflow.ChildWorkflowOptions{
				WorkflowID: fmt.Sprintf("%s_catch", workflow.GetInfo(ctx).WorkflowExecution.ID),
			}

			childCtx := workflow.WithChildOptions(ctx, opts)

			if err := workflow.ExecuteChildWorkflow(childCtx, t.catchChildWorkflowName, state.Input, state).Get(ctx, &res); err != nil {
				// Everything has failed
				logger.Error("Error calling try workflow", "error", err)
				return nil, fmt.Errorf("error calling catcg workflow: %w", err)
			}
		}

		return res, nil
	}, nil
}

func (t *TryTaskBuilder) registerTaskList(taskType string, list *model.TaskList) (childWorkflowName string, err error) {
	l := log.With().Str("task", t.GetTaskName()).Str("taskType", taskType).Logger()

	if len(*list) == 0 {
		l.Warn().Msg("No tasks detected")
		return
	}

	childWorkflowName = utils.GenerateChildWorkflowName(taskType, t.GetTaskName())

	builder, err := NewTaskBuilder(childWorkflowName, &model.DoTask{Do: list}, t.temporalWorker, t.doc)
	if err != nil {
		l.Error().Msg("Error creating the for task builder")
		err = fmt.Errorf("error creating the for task builder: %w", err)
		return
	}

	if _, err = builder.Build(); err != nil {
		l.Error().Msg("Error building for workflow")
		err = fmt.Errorf("error building for workflow: %w", err)
		return
	}

	return
}
