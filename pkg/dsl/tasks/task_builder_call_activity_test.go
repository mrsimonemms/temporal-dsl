package tasks_test

import (
	"context"
	"testing"
	"time"

	"github.com/mrsimonemms/temporal-dsl/pkg/dsl/tasks"
	"github.com/mrsimonemms/temporal-dsl/pkg/utils"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestCallActivityTaskBuilderExecute(t *testing.T) {
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	const activityName = "dslTestActivity"
	env.RegisterActivityWithOptions(func(ctx context.Context, value string) (string, error) {
		return value + "-processed", nil
	}, activity.RegisterOptions{Name: activityName})

	task := &model.CallFunction{
		Call: "activity",
		With: map[string]any{
			"name":      activityName,
			"arguments": []any{"${ .input.message }"},
			"options": map[string]any{
				"startToCloseTimeout": map[string]any{"seconds": 5},
			},
		},
	}

	b, err := tasks.NewCallActivityTaskBuilder(nil, task, "callActivity", nil)
	require.NoError(t, err)

	fn, err := b.Build()
	require.NoError(t, err)

	workflowFunc := func(ctx workflow.Context) (string, error) {
		state := utils.NewState().AddWorkflowInfo(ctx)
		state.Input = map[string]any{"message": "ping"}
		ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: time.Minute})
		result, err := fn(ctx, nil, state)
		if err != nil {
			return "", err
		}
		if result == nil {
			return "", nil
		}
		return result.(string), nil
	}

	env.ExecuteWorkflow(workflowFunc)

	var got string
	require.NoError(t, env.GetWorkflowError())
	require.NoError(t, env.GetWorkflowResult(&got))
	require.Equal(t, "ping-processed", got)
}

func TestCallActivityTaskBuilderExecuteLocal(t *testing.T) {
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	const activityName = "dslLocalActivity"
	env.RegisterActivityWithOptions(func(value string) (string, error) {
		return value + "-local", nil
	}, activity.RegisterOptions{Name: activityName})

	task := &model.CallFunction{
		Call: "activity",
		With: map[string]any{
			"name":      activityName,
			"local":     true,
			"arguments": []any{"value"},
			"localOptions": map[string]any{
				"startToCloseTimeout": map[string]any{"seconds": 2},
			},
		},
	}

	b, err := tasks.NewCallActivityTaskBuilder(nil, task, "callLocalActivity", nil)
	require.NoError(t, err)

	fn, err := b.Build()
	require.NoError(t, err)

	workflowFunc := func(ctx workflow.Context) (string, error) {
		state := utils.NewState().AddWorkflowInfo(ctx)
		ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: time.Minute})
		result, err := fn(ctx, nil, state)
		if err != nil {
			return "", err
		}
		if result == nil {
			return "", nil
		}
		return result.(string), nil
	}

	env.ExecuteWorkflow(workflowFunc)

	var got string
	require.NoError(t, env.GetWorkflowError())
	require.NoError(t, env.GetWorkflowResult(&got))
	require.Equal(t, "value-local", got)
}
