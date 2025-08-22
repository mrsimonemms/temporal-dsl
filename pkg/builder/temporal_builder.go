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

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/serverlessworkflow/sdk-go/v3/parser"
)

// Stores any activities generated from this DSL
type activities struct{}

type TemporalBuilder struct {
	Workflow *model.Workflow
	Context  context.Context
}

func (t *TemporalBuilder) GetActivities() *activities {
	return &activities{}
}

func hasMultipleWorkflows(tasks *model.TaskList) (hasMultiple bool) {
	for _, task := range *tasks {
		if do := task.AsDoTask(); do != nil {
			// Do set - treat as multiple workflows
			hasMultiple = true
		}
	}
	return
}

func (t *TemporalBuilder) workflowBuilder(tasks *model.TaskList, name *string) ([]*TemporalWorkflow, error) {
	hasMultiWorkflows := name == nil

	wfs := make([]*TemporalWorkflow, 0)

	timeout := defaultWorkflowTimeout
	if t.Workflow.Timeout != nil && t.Workflow.Timeout.Timeout != nil && t.Workflow.Timeout.Timeout.After != nil {
		timeout = ToDuration(t.Workflow.Timeout.Timeout.After)
	}

	wf := &TemporalWorkflow{
		Name:    *name,
		Tasks:   make([]TemporalWorkflowTask, 0),
		Timeout: timeout,
	}

	for _, task := range *tasks {
		var task TemporalWorkflowFunc
		var err error
		var additionalWorkflows []*TemporalWorkflow

		if hasMultiWorkflows {
			// Multiple workflows registered
		}
	}

	// Add to the list of workflows if name is set
	if !hasMultiWorkflows {
		wfs = append(wfs, wf)
	}

	return wfs, nil
}

// This converts the Serverless Workflow workflows into Temporal workflows. This
// is analogous to the Run method in impl.WorkflowRunner.
func (t *TemporalBuilder) BuildWorkflows() ([]*TemporalWorkflow, error) {
	wfs := make([]*TemporalWorkflow, 0)

	if t.Workflow.Do == nil || len(*t.Workflow.Do) == 0 {
		return nil, ErrNoTasksDefined
	}

	// The root definition can define one or more than one workflow
	// - Single workflow takes it's name from the DSL document.name
	// - Multiple workflows doesn't register document.name as a workflow
	var rootWorkflowName *string
	if !hasMultipleWorkflows(t.Workflow.Do) {
		rootWorkflowName = &t.Workflow.Document.Name
	}

	workflows, err := t.workflowBuilder(t.Workflow.Do, rootWorkflowName)
	if err != nil {
		return nil, fmt.Errorf("error building workflows: %w", err)
	}

	wfs = append(wfs, workflows...)

	return wfs, nil
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
	return &TemporalBuilder{
		Workflow: workflow,
		Context:  ctx,
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

func NewTaskBuilder(taskName string, task model.Task, workflowDef *model.Workflow) (any, error) {
	return nil, fmt.Errorf("%w: type %T for task %s", ErrUnsupportedTask, task, taskName)
}
