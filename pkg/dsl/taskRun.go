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
	"maps"

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"go.temporal.io/sdk/workflow"
)

func runChildWorkflowFn(task *model.RunTask, taskName string) (TemporalWorkflowFunc, error) {
	return func(ctx workflow.Context, data *Variables, output map[string]OutputType) error {
		logger := workflow.GetLogger(ctx)
		logger.Debug("Running a child workflow", "task", taskName)

		ctx = workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{})

		var result any
		if err := workflow.ExecuteChildWorkflow(ctx, task.Run.Workflow.Name, data.Data).Get(ctx, &result); err != nil {
			logger.Error("Error executiing child workflow", "error", err)
			return fmt.Errorf("error executiing child workflow: %w", err)
		}
		logger.Debug("Child workflow completed", "task", taskName)

		maps.Copy(output, map[string]OutputType{
			taskName: {
				Type: CallRunChildWorkflowResultType,
				Data: result,
			},
		})

		return nil
	}, nil
}

func runTaskImpl(task *model.RunTask, taskName string) (TemporalWorkflowFunc, error) {
	if task.Run.Workflow != nil {
		return runChildWorkflowFn(task, taskName)
	}

	return nil, fmt.Errorf("unsupported run task: %s", taskName)
}
