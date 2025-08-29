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

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"go.temporal.io/sdk/workflow"
)

type forkTaskOutput struct {
	name string
	data map[string]OutputType
}

// @todo(sje): handle competing forks
func forkTaskImpl(fork *model.ForkTask, task *model.TaskItem, workflowInst *Workflow) (TemporalWorkflowFunc, error) {
	childWorkflowName := GenerateChildWorkflowName("fork", task.Key)
	temporalWorkflows, err := workflowInst.workflowBuilder(fork.Fork.Branches, childWorkflowName)
	if err != nil {
		return nil, fmt.Errorf("error building forked workflow: %w", err)
	}

	n := len(temporalWorkflows)
	for _, t := range temporalWorkflows {
		n += len(t.Tasks)
	}

	return func(ctx workflow.Context, data *Variables, output map[string]OutputType) (map[string]OutputType, error) {
		logger := workflow.GetLogger(ctx)
		logger.Debug("Forking a task", "isCompeting", fork.Fork.Compete)

		chunkResultChannel := workflow.NewChannel(ctx)

		// Update the task summary to make it more readable in the UI
		ao := workflow.GetActivityOptions(ctx)
		parentTask := ao.Summary

		for _, temporalWorkflow := range temporalWorkflows {
			for _, wf := range temporalWorkflow.Tasks {
				ao.Summary = fmt.Sprintf("%s.%s", parentTask, wf.Key)
				ctx = workflow.WithActivityOptions(ctx, ao)

				workflow.Go(ctx, func(ctx workflow.Context) {
					o := make(map[string]OutputType)

					out, err := wf.Task(ctx, data, o)
					if err != nil {
						logger.Error("Error handling Temporal task", "error", err, "task", wf.Key)
						chunkResultChannel.Send(ctx, err)
						return
					}

					chunkResultChannel.Send(ctx, forkTaskOutput{
						name: wf.Key,
						data: out,
					})
				})
			}
		}

		resp := make(map[string]OutputType)

		for _, temporalWorkflow := range temporalWorkflows {
			for range temporalWorkflow.Tasks {
				var v any
				chunkResultChannel.Receive(ctx, &v)

				switch result := v.(type) {
				case error:
					if result != nil {
						return nil, result
					}
				case forkTaskOutput:
					resp[fmt.Sprintf("%s_%s", task.Key, result.name)] = OutputType{
						Type: ForkResultType,
						Data: result.data,
					}
				}
			}
		}

		return resp, nil
	}, nil
}
