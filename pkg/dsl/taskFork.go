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
	"slices"

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// A forked task allows n tasks to be run in parallel
func forkTaskImpl(fork *model.ForkTask, task *model.TaskItem, workflowInst *Workflow) (TemporalWorkflowFunc, error) {
	childWorkflowName := GenerateChildWorkflowName("fork", task.Key)
	temporalWorkflows, err := workflowInst.workflowBuilder(fork.Fork.Branches, childWorkflowName)
	if err != nil {
		return nil, fmt.Errorf("error building forked workflow: %w", err)
	}
	isCompeting := fork.Fork.Compete

	n := len(temporalWorkflows)
	for _, t := range temporalWorkflows {
		n += len(t.Tasks)
	}
	tasksCompleted := make(map[int]bool)

	return func(ctx workflow.Context, data *Variables, output map[string]OutputType) error {
		logger := workflow.GetLogger(ctx)
		logger.Debug("Forking a task", "isCompeting", isCompeting)

		// Competing forks need to be able to cancel other tasks
		cctx, cancelHandler := workflow.WithCancel(ctx)

		// Update the task summary to make it more readable in the UI
		ao := workflow.GetActivityOptions(cctx)
		parentTask := ao.Summary

		// Trigger the forked tasks
		var taskErr error
		for _, temporalWorkflow := range temporalWorkflows {
			for k, wf := range temporalWorkflow.Tasks {
				ao.Summary = fmt.Sprintf("%s.%s", parentTask, wf.Key)
				cctx = workflow.WithActivityOptions(cctx, ao)

				// Add the task
				tasksCompleted[k] = false

				workflow.Go(cctx, func(ctx workflow.Context) {
					o := make(map[string]OutputType)

					// Trigger tasks
					err := wf.Task(ctx, data, o)
					if err != nil {
						if !temporal.IsCanceledError(err) {
							logger.Error("Error handling Temporal task", "error", err, "task", wf.Key)
							taskErr = err
						}
						return
					}

					// Task has completed
					tasksCompleted[k] = true

					// Store the response - if competing, only one result is returned
					outputKey := fmt.Sprintf("%s_%s", task.Key, wf.Key)
					if isCompeting {
						outputKey = task.Key
					}
					maps.Copy(output, map[string]OutputType{
						outputKey: {
							Type: ForkResultType,
							Data: o,
						},
					})
				})
			}
		}

		// Wait for tasks to complete
		if err := workflow.Await(ctx, func() bool {
			if taskErr != nil {
				// An error has occurred
				return true
			}

			isTrue := func(v bool) bool { return v }

			mapValues := slices.Collect(maps.Values(tasksCompleted))

			if isCompeting {
				// Competing tasks - end if one has finished
				return slices.ContainsFunc(mapValues, isTrue)
			}

			// Non-competing tasks - end if all have finished
			return SliceEvery(mapValues, isTrue)
		}); err != nil {
			logger.Error("Error waiting for fork to complete", "error", err)
			return fmt.Errorf("error waiting for fork to complete: %w", err)
		}

		// Cancel any running tasks - may be competing, or error occurred
		// Make sure this happens before the error check
		logger.Debug("Cancelling other concurrent tasks")
		cancelHandler()

		if taskErr != nil {
			return fmt.Errorf("error running concurrent tasks: %w", taskErr)
		}

		return nil
	}, nil
}
