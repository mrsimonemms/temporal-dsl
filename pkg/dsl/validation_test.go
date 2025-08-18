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
	"github.com/stretchr/testify/assert"
)

// I'm not doing any tests to ensure Serverless Workflow things are supported as
// this will go away in a relatively short time
func TestValidate(t *testing.T) {
	tests := []struct {
		Name             string
		TaskList         *model.TaskList
		ValidationErrors []dsl.ValidationErrors
		Error            error
	}{
		{
			Name:     "empty tasklist",
			TaskList: &model.TaskList{},
		},
		{
			Name: "task with no metadata",
			TaskList: &model.TaskList{
				{
					Key: "test",
					Task: &model.SetTask{
						TaskBase: model.TaskBase{},
					},
				},
			},
		},
		{
			Name: "task with empty metadata",
			TaskList: &model.TaskList{
				{
					Key: "test",
					Task: &model.SetTask{
						TaskBase: model.TaskBase{
							Metadata: map[string]any{},
						},
					},
				},
			},
		},
		{
			Name: "task with no search attributes",
			TaskList: &model.TaskList{
				{
					Key: "test",
					Task: &model.SetTask{
						TaskBase: model.TaskBase{
							Metadata: map[string]any{
								dsl.MetadataSearchAttribute: map[string]*dsl.SearchAttribute{},
							},
						},
					},
				},
			},
		},
		{
			Name: "task with valid search attributes",
			TaskList: &model.TaskList{
				{
					Key: "test",
					Task: &model.SetTask{
						TaskBase: model.TaskBase{
							Metadata: map[string]any{
								dsl.MetadataSearchAttribute: map[string]*dsl.SearchAttribute{
									"example": {
										Type:  "Text",
										Value: "some text",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Name: "task with invalid search attributes",
			TaskList: &model.TaskList{
				{
					Key: "test",
					Task: &model.SetTask{
						TaskBase: model.TaskBase{
							Metadata: map[string]any{
								dsl.MetadataSearchAttribute: map[string]*dsl.SearchAttribute{
									"example": {
										Type: "invalid",
										// A nil value unsets the search attribute
										Value: nil,
									},
								},
							},
						},
					},
				},
			},
			ValidationErrors: []dsl.ValidationErrors{
				{
					Key:     "test.metadata.searchAttributes.example",
					Message: "Key: 'SearchAttribute.Type' Error:Field validation for 'Type' failed on the 'oneofci' tag",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			w := dsl.NewWorkflow(&model.Workflow{
				Do: test.TaskList,
			}, nil, "")

			vErrs, err := w.Validate()

			if test.Error == nil {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, test.Error.Error())
			}

			assert.ElementsMatch(t, vErrs, test.ValidationErrors)
		})
	}
}
