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

package dsl_test

import (
	"testing"

	"github.com/mrsimonemms/temporal-dsl/pkg/dsl"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestParseSearchAttributes(t *testing.T) {
	tests := []struct {
		Name  string
		Task  *model.TaskBase
		Vars  *dsl.Variables
		Error error
	}{
		{
			Name: "no metadata",
			Task: &model.TaskBase{
				Metadata: nil,
			},
			Error: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			testSuite := &testsuite.WorkflowTestSuite{}
			env := testSuite.NewTestWorkflowEnvironment()

			testWorkflow := func(ctx workflow.Context) error {
				// Call your function that needs workflow.Context
				return dsl.ParseSearchAttributes(ctx, test.Task, test.Vars)
			}

			env.ExecuteWorkflow(testWorkflow)

			require.True(t, env.IsWorkflowCompleted())
			require.NoError(t, env.GetWorkflowError())
		})
	}
}
