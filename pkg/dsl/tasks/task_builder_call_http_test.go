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
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/mrsimonemms/temporal-dsl/pkg/utils"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/stretchr/testify/assert"
	"go.temporal.io/sdk/testsuite"
)

func TestActivity(t *testing.T) {
	tests := []struct {
		Name             string
		Endpoint         string
		Method           string
		Body             []byte
		ResponseCode     int
		ResponseBody     any
		ExpectedBody     string
		ExpectedBodyJSON map[string]any
	}{
		{
			Name:         "Simple JSON",
			Endpoint:     "https://endpoint.com/call",
			Method:       http.MethodGet,
			ResponseCode: http.StatusOK,
			ResponseBody: map[string]any{
				"hello": "world",
			},
			ExpectedBodyJSON: map[string]any{
				"hello": "world",
			},
		},
		{
			Name:         "Simple String",
			Endpoint:     "https://endpoint.com/callString",
			Method:       http.MethodPost,
			ResponseCode: http.StatusOK,
			ResponseBody: "hello world",
			ExpectedBody: `"hello world"`,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			httpmock.Activate(t)
			t.Cleanup(httpmock.DeactivateAndReset)

			testSuite := &testsuite.WorkflowTestSuite{}

			env := testSuite.NewTestActivityEnvironment()
			env.RegisterActivity(callHTTPActivity)

			httpmock.RegisterResponder(test.Method, test.Endpoint, func(r *http.Request) (*http.Response, error) {
				return httpmock.NewJsonResponse(test.ResponseCode, test.ResponseBody)
			})

			task := &model.CallHTTP{
				With: model.HTTPArguments{
					Method:   test.Method,
					Endpoint: model.NewEndpoint(test.Endpoint),
					Body:     test.Body,
				},
			}

			state := utils.NewState()

			val, err := env.ExecuteActivity(callHTTPActivity, task, "", state)
			assert.NoError(t, err)

			var res any
			assert.NoError(t, val.Get(&res))
			assert.Equal(t, map[string]int{
				fmt.Sprintf("%s %s", test.Method, test.Endpoint): 1,
			}, httpmock.GetCallCountInfo())

			assert.Equal(t, 1, httpmock.GetTotalCallCount())
		})
	}
}
