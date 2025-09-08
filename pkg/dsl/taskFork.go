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
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// A forked task allows n tasks to be run in parallel
func forkTaskImpl(fork *model.ForkTask, task *model.TaskItem, workflowInst *Workflow) ([]*TemporalWorkflow, TemporalWorkflowFunc, error) {
	additionalWorkflows := make([]*TemporalWorkflow, 0)

	// Make each branch into it's own task list so can be run in parallel
	for _, branch := range *fork.Fork.Branches {
		branchList := &model.TaskList{
			branch,
		}

		childWorkflowName := GenerateChildWorkflowName("fork", task.Key, branch.Key)
		temporalWorkflows, err := workflowInst.workflowBuilder(branchList, childWorkflowName, workflowBuilderOpts{
			useWorkflowName: true,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("error building forked workflow: %w", err)
		}

		additionalWorkflows = append(additionalWorkflows, temporalWorkflows...)
	}
	isCompeting := fork.Fork.Compete

	return additionalWorkflows, func(ctx workflow.Context, data *Variables, output map[string]OutputType) error {
		logger := workflow.GetLogger(ctx)
		logger.Debug("Forking a task", "isCompeting", isCompeting)

		futures := map[string]CancellableFuture{}

		cancelOthers := func(passedContext workflow.Context) {
			for _, f := range futures {
				if f.Context != passedContext {
					f.Cancel()
				}
			}
		}

		// Run the child workflows in parallel
		for _, branch := range *fork.Fork.Branches {
			opts := workflow.ChildWorkflowOptions{
				WorkflowID: fmt.Sprintf("%s_fork_%s", workflow.GetInfo(ctx).WorkflowExecution.ID, branch.Key),
			}
			if isCompeting {
				// Allow cancelation without killing parent
				opts.ParentClosePolicy = enums.PARENT_CLOSE_POLICY_ABANDON
			}

			childCtx := workflow.WithChildOptions(ctx, opts)
			childCtx, cancelHandler := workflow.WithCancel(childCtx)

			childWorkflowName := GenerateChildWorkflowName("fork", task.Key, branch.Key)

			logger.Info("Triggering forked child workflow", "name", childWorkflowName)

			futures[childWorkflowName] = CancellableFuture{
				Cancel:  cancelHandler,
				Context: childCtx,
				Future:  workflow.ExecuteChildWorkflow(childCtx, childWorkflowName, data.Data),
			}
		}

		// Now they're running, wait for the results
		for taskName, w := range futures {
			var childData HTTPData
			if err := w.Future.Get(w.Context, &childData); err != nil {
				if temporal.IsCanceledError(err) {
					logger.Debug("Forked task cancelled", "task", taskName)
					return nil
				}

				logger.Error("Error forking task", "error", err, "task", taskName)
				return fmt.Errorf("error forking task: %w", err)
			}

			if isCompeting {
				logger.Info("A competing task has a winner - cancel all the other tasks", "task", taskName)
				cancelOthers(w.Context)
			}

			// Store the response - if competing, only one result is returned
			outputKey := fmt.Sprintf("%s_%s", task.Key, taskName)
			if isCompeting {
				outputKey = task.Key
			}
			maps.Copy(output, map[string]OutputType{
				outputKey: {
					Type: ForkResultType,
					Data: childData,
				},
			})
		}

		return nil
	}, nil
}
