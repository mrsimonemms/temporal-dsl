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

	"github.com/mrsimonemms/temporal-dsl/pkg/utils"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

func NewWaitTaskBuilder(temporalWorker worker.Worker, task *model.WaitTask, taskName string) (*WaitTaskBuilder, error) {
	return &WaitTaskBuilder{
		builder: builder[*model.WaitTask]{
			name:           taskName,
			task:           task,
			temporalWorker: temporalWorker,
		},
	}, nil
}

type WaitTaskBuilder struct {
	builder[*model.WaitTask]
}

func (t *WaitTaskBuilder) Build() (TemporalWorkflowFunc, error) {
	return func(ctx workflow.Context, _ any, _ *utils.State) (*utils.State, error) {
		logger := workflow.GetLogger(ctx)

		duration := utils.ToDuration(t.task.Wait)

		logger.Debug("Sleeping", "duration", duration.String())

		if err := workflow.Sleep(ctx, duration); err != nil {
			logger.Error("Error creating sleep instruction", "error", err)
			return nil, fmt.Errorf("error creating sleep: %w", err)
		}

		return nil, nil
	}, nil
}
