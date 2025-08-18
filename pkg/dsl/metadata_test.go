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
	"time"

	"github.com/mrsimonemms/temporal-dsl/pkg/dsl"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/stretchr/testify/assert"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestParseSearchAttributes(t *testing.T) {
	type attributeTest func(temporal.SearchAttributes, string) (any, bool)

	type attribute struct {
		fn    attributeTest
		value any
	}

	getBool := func(sa temporal.SearchAttributes, key string) (any, bool) {
		return sa.GetBool(temporal.NewSearchAttributeKeyBool(key))
	}

	getFloat := func(sa temporal.SearchAttributes, key string) (any, bool) {
		return sa.GetFloat64(temporal.NewSearchAttributeKeyFloat64(key))
	}

	getInt := func(sa temporal.SearchAttributes, key string) (any, bool) {
		return sa.GetInt64(temporal.NewSearchAttributeKeyInt64(key))
	}

	getKeyword := func(sa temporal.SearchAttributes, key string) (any, bool) {
		return sa.GetKeyword(temporal.NewSearchAttributeKeyKeyword(key))
	}

	getKeywordList := func(sa temporal.SearchAttributes, key string) (any, bool) {
		return sa.GetKeywordList(temporal.NewSearchAttributeKeyKeywordList(key))
	}

	getString := func(sa temporal.SearchAttributes, key string) (any, bool) {
		return sa.GetString(temporal.NewSearchAttributeKeyString(key))
	}

	getTime := func(sa temporal.SearchAttributes, key string) (any, bool) {
		return sa.GetTime(temporal.NewSearchAttributeKeyTime(key))
	}

	tests := []struct {
		Name       string
		Task       *model.TaskBase
		Vars       *dsl.Variables
		Error      error
		Attributes map[string]attribute
	}{
		{
			Name: "no metadata",
			Task: &model.TaskBase{
				Metadata: nil,
			},
			Error: nil,
		},
		{
			Name: "empty metadata",
			Task: &model.TaskBase{
				Metadata: map[string]any{},
			},
			Error: nil,
		},
		{
			Name: "empty search parameters",
			Task: &model.TaskBase{
				Metadata: map[string]any{
					dsl.MetadataSearchAttribute: map[string]*dsl.SearchAttribute{},
				},
			},
			Error: nil,
		},
		{
			Name: "complete search parameters",
			Task: &model.TaskBase{
				Metadata: map[string]any{
					dsl.MetadataSearchAttribute: map[string]*dsl.SearchAttribute{
						"bool-setter-true": {
							Type:  dsl.SearchAttributeBooleanType,
							Value: true,
						},
						"bool-setter-false": {
							Type:  dsl.SearchAttributeBooleanType,
							Value: false,
						},
						"bool-str-setter-true": {
							Type:  dsl.SearchAttributeBooleanType,
							Value: "true",
						},
						"bool-str-setter-false": {
							Type:  dsl.SearchAttributeBooleanType,
							Value: "false",
						},
						"datetime-string-setter": {
							Type:  dsl.SearchAttributeDateTimeType,
							Value: "2025-04-21T09:18:00Z",
						},
						"date-string-setter": {
							Type:  dsl.SearchAttributeDateTimeType,
							Value: "2025-03-01T00:00:00.000Z",
						},
						"datetime-setter": {
							Type:  dsl.SearchAttributeDateTimeType,
							Value: time.Date(2025, 4, 21, 9, 18, 0, 0, time.UTC),
						},
						"date-setter": {
							Type:  dsl.SearchAttributeDateTimeType,
							Value: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
						},
						"int-int-setter": {
							Type:  dsl.SearchAttributeIntType,
							Value: int(27),
						},
						"int-int32-setter": {
							Type:  dsl.SearchAttributeIntType,
							Value: int32(43),
						},
						"int-int64-setter": {
							Type:  dsl.SearchAttributeIntType,
							Value: int64(4097),
						},
						"int-float32-setter": {
							Type:  dsl.SearchAttributeIntType,
							Value: float32(41),
						},
						"int-float64-setter": {
							Type:  dsl.SearchAttributeIntType,
							Value: float32(598),
						},
						"int-string-setter": {
							Type:  dsl.SearchAttributeIntType,
							Value: "239",
						},
						"float-int-setter": {
							Type:  dsl.SearchAttributeDoubleType,
							Value: int(27),
						},
						"float-int32-setter": {
							Type:  dsl.SearchAttributeDoubleType,
							Value: int32(43),
						},
						"float-int64-setter": {
							Type:  dsl.SearchAttributeDoubleType,
							Value: int64(4097),
						},
						"float-float32-setter": {
							Type:  dsl.SearchAttributeDoubleType,
							Value: float32(41),
						},
						"float-float64-setter": {
							Type:  dsl.SearchAttributeDoubleType,
							Value: float32(598),
						},
						"float-string-setter": {
							Type:  dsl.SearchAttributeDoubleType,
							Value: "239",
						},
						"keyword-setter": {
							Type:  dsl.SearchAttributeKeywordType,
							Value: "some-keyword",
						},
						"keyword-list-setter": {
							Type:  dsl.SearchAttributeKeywordListType,
							Value: []string{"some-keyword1", "some-keyword2"},
						},
						"text-setter": {
							Type:  dsl.SearchAttributeTextType,
							Value: "some-text",
						},
					},
				},
			},
			Attributes: map[string]attribute{
				"bool-setter-true": {
					fn:    getBool,
					value: true,
				},
				"bool-setter-false": {
					fn:    getBool,
					value: false,
				},
				"bool-str-setter-true": {
					fn:    getBool,
					value: true,
				},
				"bool-str-setter-false": {
					fn:    getBool,
					value: false,
				},
				"datetime-string-setter": {
					value: time.Date(2025, 4, 21, 9, 18, 0, 0, time.UTC),
					fn:    getTime,
				},
				"date-string-setter": {
					value: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
					fn:    getTime,
				},
				"datetime-setter": {
					value: time.Date(2025, 4, 21, 9, 18, 0, 0, time.UTC),
					fn:    getTime,
				},
				"date-setter": {
					value: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
					fn:    getTime,
				},
				"int-int-setter": {
					fn:    getInt,
					value: int64(27),
				},
				"int-int32-setter": {
					fn:    getInt,
					value: int64(43),
				},
				"int-int64-setter": {
					fn:    getInt,
					value: int64(4097),
				},
				"int-float32-setter": {
					fn:    getInt,
					value: int64(41),
				},
				"int-float64-setter": {
					fn:    getInt,
					value: int64(598),
				},
				"int-string-setter": {
					fn:    getInt,
					value: int64(239),
				},
				"float-int-setter": {
					fn:    getFloat,
					value: float64(27),
				},
				"float-int32-setter": {
					fn:    getFloat,
					value: float64(43),
				},
				"float-int64-setter": {
					fn:    getFloat,
					value: float64(4097),
				},
				"float-float32-setter": {
					fn:    getFloat,
					value: float64(41),
				},
				"float-float64-setter": {
					fn:    getFloat,
					value: float64(598),
				},
				"float-string-setter": {
					fn:    getFloat,
					value: float64(239),
				},
				"keyword-list-setter": {
					value: []string{"some-keyword1", "some-keyword2"},
					fn:    getKeywordList,
				},
				"keyword-setter": {
					value: "some-keyword",
					fn:    getKeyword,
				},
				"text-setter": {
					value: "some-text",
					fn:    getString,
				},
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
				err := dsl.ParseSearchAttributes(ctx, test.Task, test.Vars)

				if test.Error == nil {
					assert.NoError(t, err)
				} else {
					assert.EqualError(t, err, test.Error.Error())
				}

				sa := workflow.GetTypedSearchAttributes(ctx)

				assert.Equal(t, len(test.Attributes), sa.Size())

				for k, attr := range test.Attributes {
					value, exists := attr.fn(sa, k)
					assert.Equal(t, attr.value, value, k)
					assert.True(t, exists, k)
				}

				return nil
			}

			// Trigger
			env.ExecuteWorkflow(testWorkflow)

			assert.True(t, env.IsWorkflowCompleted())
			assert.NoError(t, env.GetWorkflowError())
		})
	}
}
