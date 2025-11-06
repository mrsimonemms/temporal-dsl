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
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

func NewRunTaskBuilder(
	temporalWorker worker.Worker,
	task *model.RunTask,
	taskName string,
) (*RunTaskBuilder, error) {
	return &RunTaskBuilder{
		builder: builder[*model.RunTask]{
			name:           taskName,
			task:           task,
			temporalWorker: temporalWorker,
		},
	}, nil
}

// Ensure the task meets the TaskBuilder type
var _ TaskBuilder = &RunTaskBuilder{}

type RunTaskBuilder struct {
	builder[*model.RunTask]
}

func (t *RunTaskBuilder) Build() (TemporalWorkflowFunc, error) {
	return func(ctx workflow.Context, input any, state *utils.State) (any, error) {
		logger := workflow.GetLogger(ctx)

		if t.task.Run.Workflow == nil {
			return nil, fmt.Errorf("unsupported run task: %s", t.GetTaskName())
		}

		if t.task.Run.Await == nil {
			// Default to true
			t.task.Run.Await = utils.Ptr(true)
		}

		logger.Debug("Run await status", "await", *t.task.Run.Await, "task", t.GetTaskName())

		return t.runWorkflow(ctx, input, state)
	}, nil
}

func (t *RunTaskBuilder) runWorkflow(ctx workflow.Context, input any, state *utils.State) (any, error) {
	logger := workflow.GetLogger(ctx)
	logger.Debug("Running a child workflow", "task", t.GetTaskName())

	await := *t.task.Run.Await

	opts := workflow.ChildWorkflowOptions{}
	if !await {
		opts.ParentClosePolicy = enums.PARENT_CLOSE_POLICY_ABANDON
	}

	ctx = workflow.WithChildOptions(ctx, opts)

	future := workflow.ExecuteChildWorkflow(ctx, t.task.Run.Workflow.Name, input, state)

	if !await {
		logger.Warn("Not waiting for child workspace response", "task", t.GetTaskName())
		return nil, nil
	}

	var res any
	if err := future.Get(ctx, &res); err != nil {
		logger.Error("Error executiing child workflow", "error", err)
		return nil, fmt.Errorf("error executiing child workflow: %w", err)
	}
	logger.Debug("Child workflow completed", "task", t.GetTaskName())

	return res, nil
}
