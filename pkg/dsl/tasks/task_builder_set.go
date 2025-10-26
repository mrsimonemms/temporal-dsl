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

	"github.com/mrsimonemms/temporal-dsl/pkg/utils"
	swUtils "github.com/serverlessworkflow/sdk-go/v3/impl/utils"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

func NewSetTaskBuilder(temporalWorker worker.Worker, task *model.SetTask, taskName string) (*SetTaskBuilder, error) {
	return &SetTaskBuilder{
		builder: builder[*model.SetTask]{
			name:           taskName,
			task:           task,
			temporalWorker: temporalWorker,
		},
	}, nil
}

type SetTaskBuilder struct {
	builder[*model.SetTask]
}

func (t *SetTaskBuilder) Build() (TemporalWorkflowFunc, error) {
	return func(ctx workflow.Context, input any, state *utils.State) (*utils.State, error) {
		setObject := swUtils.DeepClone(t.task.Set)

		result, err := utils.TraverseAndEvaluateObj(
			model.NewObjectOrRuntimeExpr(setObject),
			state,
			func(f func() (any, error)) (any, error) {
				return t.sideEffectWrapper(ctx, f)
			},
		)
		if err != nil {
			return nil, fmt.Errorf("error parsing set data: %w", err)
		}

		// Add the newly set data into the state
		return state.BulkAdd(result), nil
	}, nil
}

// sideEffectWrapper creates a wrapper function for the Runtime Expression traversal to ensure that
// the generated values are set deterministically. For many things, this might be considered overkill
// as input/envvars/state are likely to be determinstic. However, as this also supports things like
// generation of UUIDs, there could be non-deterministic values being set.
func (t *SetTaskBuilder) sideEffectWrapper(ctx workflow.Context, fn func() (any, error)) (any, error) {
	var val any
	var sideEffectErr error
	err := workflow.SideEffect(ctx, func(ctx workflow.Context) any {
		res, err := fn()
		if err != nil {
			sideEffectErr = err
			return nil
		}
		return res
	}).Get(&val)
	if err != nil {
		return nil, fmt.Errorf("error running side effect: %w", err)
	}
	if sideEffectErr != nil {
		return nil, fmt.Errorf("error running runtime expression: %w", sideEffectErr)
	}

	return val, nil
}
