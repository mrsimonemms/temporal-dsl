package runtime

import (
	"context"
	"fmt"

	"github.com/serverlessworkflow/sdk-go/v3/model"
)

type ExecutionCtx struct {
	Data map[string]any
}

type Engine struct{ reg *Registry }

func NewEngine(reg *Registry) *Engine { return &Engine{reg: reg} }

func (e *Engine) Run(ctx context.Context, wf *model.Workflow, input map[string]any) (any, error) {
	exec := &ExecutionCtx{Data: input}

	if wf.Do == nil || len(wf.Do) == 0 {
		return nil, fmt.Errorf("workflow has no 'do' steps")
	}
	var last any
	for _, step := range wf.Do {
		switch {
		case step.Call != nil && step.Call.HTTP != nil:
			task := map[string]any{"call": map[string]any{"http": step.Call.HTTP}}
			execor, _ := e.reg.Get("call.http")
			var err error
			last, err = execor.Execute(exec, task)
			if err != nil {
				return nil, err
			}
		case step.Run != nil && step.Run.Process != nil:
			task := map[string]any{"run": map[string]any{"process": step.Run.Process}}
			execor, _ := e.reg.Get("run.process")
			var err error
			last, err = execor.Execute(exec, task)
			if err != nil {
				return nil, err
			}
		default:
			// not implemented yet
		}
	}
	return last, nil
}
