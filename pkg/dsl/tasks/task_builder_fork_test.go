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

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

// childCompletesAfter sleeps then returns a value (or a CancelledError if cancelled).
func childCompletesAfter(ctx workflow.Context, d time.Duration, name string) (any, error) {
	if err := workflow.Sleep(ctx, d); err != nil {
		return nil, err
	}
	return name, nil
}

// childErrorsAfter always returns an error after a short delay.
func childErrorsAfter(ctx workflow.Context, d time.Duration, msg string) (any, error) {
	_ = workflow.Sleep(ctx, d)
	return nil, errors.New(msg)
}

func forkTaskBuilder(compete bool) *ForkTaskBuilder {
	return &ForkTaskBuilder{
		builder: builder[*model.ForkTask]{
			task: &model.ForkTask{
				Fork: model.ForkTaskConfiguration{
					Compete:  compete,
					Branches: &model.TaskList{},
				},
			},
		},
	}
}

type childSpec struct {
	key  string
	name string
	run  func(ctx workflow.Context, input any, state map[string]any) (any, error)
}

func TestForkTaskBuilder_ForkModes(t *testing.T) {
	cases := []struct {
		name        string
		compete     bool
		children    []childSpec
		expectError bool
	}{
		{
			name:    "non-competing: waits for all",
			compete: false,
			children: []childSpec{
				{
					key:  "a",
					name: "fork-a",
					run: func(ctx workflow.Context, _ any, _ map[string]any) (any, error) {
						return childCompletesAfter(ctx, 10*time.Millisecond, "a")
					},
				},
				{
					key:  "b",
					name: "fork-b",
					run: func(ctx workflow.Context, _ any, _ map[string]any) (any, error) {
						return childCompletesAfter(ctx, 20*time.Millisecond, "b")
					},
				},
			},
		},
		{
			name:    "competing: first finisher cancels others",
			compete: true,
			children: []childSpec{
				{
					key:  "fast",
					name: "fork-fast",
					run: func(ctx workflow.Context, _ any, _ map[string]any) (any, error) {
						return childCompletesAfter(ctx, 5*time.Millisecond, "fast")
					},
				},
				{
					key:  "slow",
					name: "fork-slow",
					run: func(ctx workflow.Context, _ any, _ map[string]any) (any, error) {
						// Long sleep to ensure it is the loser and gets cancelled.
						return childCompletesAfter(ctx, 5*time.Second, "slow")
					},
				},
			},
		},
		{
			name:    "non-competing: error propagates",
			compete: false,
			children: []childSpec{
				{
					key:  "ok",
					name: "fork-ok",
					run: func(ctx workflow.Context, _ any, _ map[string]any) (any, error) {
						return childCompletesAfter(ctx, 5*time.Millisecond, "ok")
					},
				},
				{
					key:  "boom",
					name: "fork-boom",
					run: func(ctx workflow.Context, _ any, _ map[string]any) (any, error) {
						return childErrorsAfter(ctx, 5*time.Millisecond, "kaboom")
					},
				},
			},
			expectError: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var ts testsuite.WorkflowTestSuite
			env := ts.NewTestWorkflowEnvironment()

			// Build the parent workflow func from exec() using forked task descriptors.
			b := forkTaskBuilder(tc.compete)
			forked := make([]*forkedTask, 0, len(tc.children))
			for _, ch := range tc.children {
				forked = append(forked, &forkedTask{
					task:              &model.TaskItem{Key: ch.key},
					childWorkflowName: ch.name,
				})
			}
			parent, err := b.exec(forked)
			if err != nil {
				t.Fatalf("exec() returned error: %v", err)
			}

			// Register parent under a stable name and all children under their provided names.
			env.RegisterWorkflowWithOptions(parent, workflow.RegisterOptions{Name: "parent"})
			for _, ch := range tc.children {
				env.RegisterWorkflowWithOptions(ch.run, workflow.RegisterOptions{Name: ch.name})
			}

			// Execute
			env.ExecuteWorkflow("parent", nil, map[string]any{})

			// Assertions
			if !env.IsWorkflowCompleted() {
				t.Fatalf("parent workflow did not complete")
			}
			if tc.expectError {
				if env.GetWorkflowError() == nil {
					t.Fatalf("expected parent to return error, got nil")
				}
			} else if err := env.GetWorkflowError(); err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
		})
	}
}
