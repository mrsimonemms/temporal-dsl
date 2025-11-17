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

package tasks_test

import (
	"testing"
	"time"

	"github.com/mrsimonemms/temporal-dsl/pkg/dsl/tasks"
	"github.com/mrsimonemms/temporal-dsl/pkg/utils"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/stretchr/testify/assert"
	"go.temporal.io/sdk/testsuite"
)

func TestWaitTaskBuilder(t *testing.T) {
	tests := []struct {
		Name     string
		Duration model.DurationInline
	}{
		{
			Name: "10 second delay",
			Duration: model.DurationInline{
				Seconds: 10,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			var s testsuite.WorkflowTestSuite
			env := s.NewTestWorkflowEnvironment()

			start := time.Now().UTC()
			env.SetStartTime(start)

			dur := &model.Duration{
				Value: test.Duration,
			}

			w, err := tasks.NewWaitTaskBuilder(nil, &model.WaitTask{
				Wait: dur,
			}, test.Name, nil)
			assert.NoError(t, err)

			wf, err := w.Build()
			assert.NoError(t, err)

			env.RegisterWorkflow(wf)

			env.ExecuteWorkflow(wf, nil, nil)

			assert.NoError(t, env.GetWorkflowError())

			got := env.Now().UTC()
			want := start.Add(utils.ToDuration(dur))

			assert.True(t, got.Equal(want))
		})
	}
}
