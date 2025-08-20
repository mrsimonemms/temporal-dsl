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

package builder

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/serverlessworkflow/sdk-go/v3/impl"
	swContext "github.com/serverlessworkflow/sdk-go/v3/impl/ctx"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/serverlessworkflow/sdk-go/v3/parser"
)

// Stores any activities generated from this DSL
type activities struct{}

type TemporalBuilder struct {
	Workflow  *model.Workflow
	Context   context.Context
	RunnerCtx swContext.WorkflowContext
}

func (t *TemporalBuilder) GetActivities() *activities {
	return &activities{}
}

// This converts the Serverless Workflow workflows into Temporal workflows. This
// is analogous to the Run method in impl.WorkflowRunner.
func (t *TemporalBuilder) BuildWorkflows() ([]*TemporalWorkflow, error) {
	wfs := make([]*TemporalWorkflow, 0)

	doRunner, err := impl.NewDoTaskRunner(t.Workflow.Do)
	if err != nil {
		return nil, fmt.Errorf("error building workflows: %w", err)
	}

	fmt.Printf("%+v\n", doRunner)

	return wfs, nil
}

func (t *TemporalBuilder) GetWorkflowCtx() swContext.WorkflowContext {
	return t.RunnerCtx
}

func (t *TemporalBuilder) GetWorkflowDef() *model.Workflow {
	return t.Workflow
}

// NewTemporalBuilder creates an instance of the TemporalBuilder type. It has
// a different interface from the Serverless Workflow's impl.WorkflowRunner type
// because the workflow logic runs in Temporal. However, it aims to follow the
// conventions of Serverless Workflow.
//
// This builder is designed to generate the Temporal code from the DSL
func NewTemporalBuilder(ctx context.Context, workflow *model.Workflow) (*TemporalBuilder, error) {
	wfContext, err := swContext.NewWorkflowContext(workflow)
	if err != nil {
		return nil, fmt.Errorf("error creating workflow context: %w", err)
	}

	return &TemporalBuilder{
		Workflow:  workflow,
		Context:   swContext.WithWorkflowContext(ctx, wfContext),
		RunnerCtx: wfContext,
	}, nil
}

func LoadWorkflowFile(file string) (*model.Workflow, error) {
	data, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		return nil, fmt.Errorf("error loading file: %w", err)
	}

	wf, err := parser.FromYAMLSource(data)
	if err != nil {
		return nil, fmt.Errorf("error loading yaml: %w", err)
	}

	// Only support dsl v1.0.0 - we will likely support later versions
	if dsl := wf.Document.DSL; dsl != "1.0.0" {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedDSL, dsl)
	}

	return wf, nil
}
