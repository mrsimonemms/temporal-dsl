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

func setSwitchImpl(task *model.SwitchTask, key string) (TemporalWorkflowFunc, error) {
	hasDefault := false
	for i, switchItem := range task.Switch {
		for name, item := range switchItem {
			if item.When == nil {
				if hasDefault {
					return nil, fmt.Errorf("multiple switch statements without when: %s.%d.%s", key, i, name)
				}
				hasDefault = true
			}
		}
	}

	return func(ctx workflow.Context, data *Variables, output map[string]OutputType) (map[string]OutputType, error) {
		logger := workflow.GetLogger(ctx)

		for _, switchItem := range task.Switch {
			for name, item := range switchItem {
				logger.Debug("Checking if we should run this switch statement", "taskName", key, "switchCondition", name)

				if toRun, err := CheckIfStatement(item.When, data); err != nil {
					logger.Error("Error checking switch statement's when", "error", err)
					return nil, err
				} else if !toRun {
					logger.Debug("Skipping switch statement task", "taskName", key, "switchCondition", name)
					continue
				}

				then := item.Then
				if then == nil || then.IsTermination() {
					logger.Debug("Skipping task as then is termination or not set")
					return nil, nil
				}

				logger.Info("Executing switch statement's task as a child workflow", "taskName", key, "switchCondition", name)
				var result map[string]OutputType
				if err := workflow.ExecuteChildWorkflow(ctx, then.Value, data).Get(ctx, &result); err != nil {
					logger.Error("Error executing child workflow")
					return nil, err
				}

				// Stop it executing anything else
				return map[string]OutputType{
					key: {
						Type: CallHTTPResultType,
						Data: result,
					},
				}, nil
			}
		}

		return nil, nil
	}, nil
}
