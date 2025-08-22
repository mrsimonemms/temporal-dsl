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

package builder_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mrsimonemms/temporal-dsl/pkg/builder"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/stretchr/testify/assert"
)

func TestNewTemporalBuilder(t *testing.T) {
	tests := []struct {
		Name        string
		Context     context.Context
		Workflow    *model.Workflow
		ExpectError bool
	}{
		{
			Name:    "successful creation with valid workflow",
			Context: context.Background(),
			Workflow: &model.Workflow{
				Document: model.Document{
					DSL: "1.0.0",
				},
				Do: &model.TaskList{}, // Empty tasks for basic testing
			},
			ExpectError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			r, err := builder.NewTemporalBuilder(test.Context, test.Workflow)

			if test.ExpectError {
				assert.Error(t, err)
				assert.Nil(t, r)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, r)
				assert.Equal(t, test.Workflow, r.Workflow)
				assert.NotNil(t, r.Context)
			}
		})
	}
}

func TestLoadWorkflowFile(t *testing.T) {
	tests := []struct {
		Name        string
		Content     string
		Error       error
		ExpectError bool
	}{
		{
			Name: "Load valid workflow file",
			Content: `document:
  dsl: 1.0.0
  namespace: default
  name: test
  version: 0.0.1
do:
  - step:
      set:
        hello: world`,
		},
		{
			Name: "Invalid DSL version",
			Content: `document:
  dsl: 0.9.0
  namespace: default
  name: test
  version: 0.0.1
do:
  - step:
      set:
        hello: world`,
			Error:       builder.ErrUnsupportedDSL,
			ExpectError: true,
		},
		{
			Name:        "Invalid YAML",
			Content:     `invalid content: [`,
			ExpectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "workflow_test")
			assert.NoError(t, err)
			defer func() {
				assert.NoError(t, os.RemoveAll(tmpDir))
			}()

			filePath := filepath.Join(tmpDir, "dsl.yaml")
			err = os.WriteFile(filePath, []byte(test.Content), 0o600)
			assert.NoError(t, err)

			workflow, err := builder.LoadWorkflowFile(filePath)
			if test.ExpectError {
				assert.Error(t, err)
				assert.Nil(t, workflow)

				if test.Error != nil {
					assert.ErrorIs(t, err, test.Error)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, workflow)
			}
		})
	}
}
