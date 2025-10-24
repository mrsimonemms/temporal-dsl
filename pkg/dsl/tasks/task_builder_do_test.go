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
	"errors"
	"testing"
	"time"

	"github.com/mrsimonemms/temporal-dsl/pkg/utils"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/stretchr/testify/assert"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

// doTaskBuilder creates a DoTaskBuilder that we can use to call workflowExecutor directly.
func doTaskBuilder(name string) *DoTaskBuilder {
	return &DoTaskBuilder{
		builder: builder[*model.DoTask]{
			name: name,
			doc:  &model.Workflow{}, // no timeout -> should use defaultWorkflowTimeout
		},
	}
}

// wfOK asserts the AO summary matches `wantSummary` and that the timeout is the package default.
func wfOK(wantSummary string) func(ctx workflow.Context, _ any, _ *utils.State) (*utils.State, error) {
	return func(ctx workflow.Context, _ any, _ *utils.State) (*utils.State, error) {
		ao := workflow.GetActivityOptions(ctx)
		if ao.Summary != wantSummary {
			return nil, errors.New("summary not set correctly: got " + ao.Summary + ", want " + wantSummary)
		}
		if ao.StartToCloseTimeout != defaultWorkflowTimeout {
			return nil, errors.New("unexpected timeout; got " + ao.StartToCloseTimeout.String())
		}
		return nil, nil
	}
}

// wfErr returns an error to ensure the parent stops the sequence and propagates the error.
func wfErr(msg string) func(ctx workflow.Context, _ any, _ *utils.State) (*utils.State, error) {
	return func(ctx workflow.Context, _ any, _ *utils.State) (*utils.State, error) {
		// small sleep to simulate work (and allow cancellation semantics if needed in future)
		_ = workflow.Sleep(ctx, 5*time.Millisecond)
		return nil, errors.New(msg)
	}
}

func TestDoTaskBuilder_WorkflowExecutor_StopsOnFirstError(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()

	parentName := "parent-error"
	b := doTaskBuilder(parentName)

	// First task errors; second must never run. We'll detect "ran" by returning a specific error if it executes.
	secondRanErr := errors.New("second task should not have run")

	tasks := []workflowFunc{
		{
			Name: "first",
			Func: wfErr("boom"),
		},
		{
			Name: "second",
			Func: func(ctx workflow.Context, _ any, _ *utils.State) (*utils.State, error) {
				return nil, secondRanErr
			},
		},
	}

	parent := b.workflowExecutor(tasks)

	env.RegisterWorkflowWithOptions(parent, workflow.RegisterOptions{Name: parentName})
	env.ExecuteWorkflow(parentName, nil, map[string]any{})

	if !env.IsWorkflowCompleted() {
		t.Fatalf("workflow did not complete")
	}

	err := env.GetWorkflowError()
	if err == nil {
		t.Fatalf("expected error from first task, got nil")
	}
	if errors.Is(err, secondRanErr) {
		t.Fatalf("second task executed; sequence did not stop on first error")
	}
}

func TestDoTaskBuilder_WorkflowExecutor_SetsSummaryAndReturnsHelloWorld(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()

	parentName := "parent-ok"
	b := doTaskBuilder(parentName)

	tasks := []workflowFunc{
		{
			Name: "task-1",
			Func: wfOK("task-1"),
		},
		{
			Name: "task-2",
			Func: wfOK("task-2"),
		},
	}

	parent := b.workflowExecutor(tasks)

	env.RegisterWorkflowWithOptions(parent, workflow.RegisterOptions{Name: parentName})
	env.ExecuteWorkflow(parentName, nil, map[string]any{})

	assert.True(t, env.IsWorkflowCompleted())
	assert.NoError(t, env.GetWorkflowError())

	var result any
	assert.NoError(t, env.GetWorkflowResult(&result))
	expected := map[string]any{}
	assert.Equal(t, result, expected)
}
