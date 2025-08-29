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

func waitTaskImpl(task *model.WaitTask) TemporalWorkflowFunc {
	return func(ctx workflow.Context, data *Variables, output map[string]OutputType) (map[string]OutputType, error) {
		logger := workflow.GetLogger(ctx)

		duration := ToDuration(task.Wait)

		logger.Debug("Sleeping", "duration", duration.String())

		if err := workflow.Sleep(ctx, duration); err != nil {
			return nil, fmt.Errorf("error sleeping: %w", err)
		}

		return nil, nil
	}
}
