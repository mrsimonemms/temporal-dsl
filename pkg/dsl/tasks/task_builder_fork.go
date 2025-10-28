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
	"time"

	"github.com/mrsimonemms/temporal-dsl/pkg/utils"
	"github.com/rs/zerolog/log"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

func NewForkTaskBuilder(
	temporalWorker worker.Worker,
	task *model.ForkTask,
	taskName string,
	doc *model.Workflow,
) (*ForkTaskBuilder, error) {
	return &ForkTaskBuilder{
		builder: builder[*model.ForkTask]{
			doc:            doc,
			name:           taskName,
			task:           task,
			temporalWorker: temporalWorker,
		},
	}, nil
}

type ForkTaskBuilder struct {
	builder[*model.ForkTask]
}

type forkedTask struct {
	task              *model.TaskItem
	childWorkflowName string
}

func (t *ForkTaskBuilder) Build() (TemporalWorkflowFunc, error) {
	forkedTasks := make([]*forkedTask, 0)

	for _, branch := range *t.task.Fork.Branches {
		childWorkflowName := utils.GenerateChildWorkflowName("fork", t.GetTaskName(), branch.Key)

		forkedTasks = append(forkedTasks, &forkedTask{
			task:              branch,
			childWorkflowName: childWorkflowName,
		})

		if d := branch.AsDoTask(); d == nil {
			// Single task - register this as a single task workflow
			log.Debug().Str("task", branch.Key).Msg("Registering single task workflow")
			branch = &model.TaskItem{
				Key: childWorkflowName,
				Task: &model.DoTask{
					Do: &model.TaskList{branch},
				},
			}
		}

		builder, err := NewTaskBuilder(childWorkflowName, branch.Task, t.temporalWorker, t.doc)
		if err != nil {
			log.Error().Err(err).Msg("Error creating the forked task builder")
			return nil, fmt.Errorf("error creating the forked task builder: %w", err)
		}

		if _, err := builder.Build(); err != nil {
			log.Error().Err(err).Msg("Error building forked workflow")
			return nil, fmt.Errorf("error building forked workflow: %w", err)
		}
	}

	return t.exec(forkedTasks)
}

// @todo(sje): figure out the input and output
func (t *ForkTaskBuilder) exec(forkedTasks []*forkedTask) (TemporalWorkflowFunc, error) {
	return func(ctx workflow.Context, input any, state *utils.State) (any, error) {
		isCompeting := t.task.Fork.Compete

		logger := workflow.GetLogger(ctx)
		logger.Debug("Forking a task", "isCompeting", isCompeting)

		ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: time.Minute,
		})

		futures := &utils.CancellableFutures{}

		// Run the child workflows in parallel
		for _, branch := range forkedTasks {
			opts := workflow.ChildWorkflowOptions{
				WorkflowID: fmt.Sprintf("%s_fork_%s", workflow.GetInfo(ctx).WorkflowExecution.ID, branch.task.Key),
			}
			if isCompeting {
				// Allow cancellation without killing parent
				opts.ParentClosePolicy = enums.PARENT_CLOSE_POLICY_ABANDON
			}

			childCtx := workflow.WithChildOptions(ctx, opts)
			childCtx, cancelHandler := workflow.WithCancel(childCtx)

			logger.Info("Triggering forked child workflow", "name", branch.childWorkflowName)

			futures.Add(branch.childWorkflowName, utils.CancellableFuture{
				Cancel:  cancelHandler,
				Context: childCtx,
				Future:  workflow.ExecuteChildWorkflow(childCtx, branch.childWorkflowName, input, state),
			})
		}

		// Now they're running, wait for the results
		var replyErr error
		hasReplied := make([]bool, futures.Length())
		var winningCtx workflow.Context

		i := 0
		for taskName, w := range futures.List() {
			// Get the replies in parallel as the "winner" may be last
			workflow.Go(w.Context, func(ctx workflow.Context) {
				var childData any
				if err := w.Future.Get(ctx, &childData); err != nil {
					if temporal.IsCanceledError(err) {
						logger.Debug("Forked task cancelled", "task", taskName)
						return
					}

					logger.Error("Error forking task", "error", err, "task", taskName)
					replyErr = fmt.Errorf("error forking task: %w", err)
				}

				hasReplied[i] = true

				// Always add non-competing data to the output
				if isCompeting && winningCtx == nil {
					logger.Debug(
						"Winner declared",
						"childWorkflowID",
						workflow.GetChildWorkflowOptions(ctx).WorkflowID,
					)

					winningCtx = ctx
				}

				i++
			})
		}

		// Wait for the concurrent tasks to complete
		if err := workflow.Await(ctx, func() bool {
			if replyErr != nil {
				return true
			}

			predicate := func(v bool) bool { return v }

			if isCompeting {
				return winningCtx != nil
			}

			return utils.SliceEvery(hasReplied, predicate)
		}); err != nil {
			logger.Error("Error waiting for forked tasks to complete", "error", err)
			return nil, fmt.Errorf("error waiting for forked tasks to complete: %w", err)
		}

		logger.Debug("Forked task has completed")

		if replyErr != nil {
			return nil, replyErr
		}

		if isCompeting {
			logger.Debug("Cancelling other forked workflows")
			futures.CancelOthers(winningCtx)
		}

		return nil, nil
	}, nil
}
