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
	"testing"

	"github.com/mrsimonemms/temporal-dsl/pkg/utils"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/stretchr/testify/assert"
	"go.temporal.io/sdk/temporal"
)

// mockTask implements model.Task for testing purposes
type mockTask struct {
	base *model.TaskBase
}

func (m *mockTask) GetBase() *model.TaskBase {
	return m.base
}

func TestShouldRun(t *testing.T) {
	type testCase struct {
		name          string
		ifExpr        string
		expectedRun   bool
		expectError   bool
		errorContains string
	}

	tests := []testCase{
		{
			name:        "No If statement returns true",
			ifExpr:      "",
			expectedRun: true,
		},
		{
			name:        "Boolean true returns true",
			ifExpr:      "true",
			expectedRun: true,
		},
		{
			name:        "Boolean false returns false",
			ifExpr:      "false",
			expectedRun: false,
		},
		{
			name:        "String TRUE returns true",
			ifExpr:      "TRUE",
			expectedRun: true,
		},
		{
			name:        "String '1' returns true",
			ifExpr:      "1",
			expectedRun: true,
		},
		{
			name:        "String FALSE returns false",
			ifExpr:      "FALSE",
			expectedRun: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var ifField *model.RuntimeExpression
			if tc.ifExpr != "" {
				ifField = &model.RuntimeExpression{Value: tc.ifExpr}
			}

			task := &mockTask{base: &model.TaskBase{If: ifField}}
			b := builder[*mockTask]{
				task: task,
			}

			result, err := b.ShouldRun(&utils.State{})

			if tc.expectError {
				assert.Error(t, err)
				var appErr *temporal.ApplicationError
				assert.ErrorAs(t, err, &appErr)
				assert.Contains(t, err.Error(), tc.errorContains)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tc.expectedRun, result)
		})
	}
}
