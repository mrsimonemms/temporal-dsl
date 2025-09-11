package runtime

import "fmt"

type TaskExecutor interface {
	Kind() string
	Execute(ctx *ExecutionCtx, task map[string]any) (any, error)
}

type Registry struct {
	byKind map[string]TaskExecutor
}

func NewRegistry(executors ...TaskExecutor) *Registry {
	r := &Registry{byKind: map[string]TaskExecutor{}}
	for _, e := range executors {
		r.byKind[e.Kind()] = e
	}
	return r
}

func (r *Registry) Get(kind string) (TaskExecutor, error) {
	e, ok := r.byKind[kind]
	if !ok {
		return nil, fmt.Errorf("no executor registered for %q", kind)
	}
	return e, nil
}
