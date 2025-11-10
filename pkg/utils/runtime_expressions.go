/*
 * Copyright 2025 Temporal DSL authors <https://github.com/mrsimonemms/temporal-dsl/graphs/contributors>
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

package utils

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/itchyny/gojq"
	"github.com/serverlessworkflow/sdk-go/v3/model"
)

type ExpressionWrapperFunc func(func() (any, error)) (any, error)

type jqFunc struct {
	Name    string                         // Becomes the name of the function to use (eg, ${ uuid })
	MinArgs int                            // Minimum number of args
	MaxArgs int                            // Maximum number of args
	Func    func(vars any, args []any) any // The function - receives the variables and arguments
}

// List of functions that are available as a function
var jqFuncs []jqFunc = []jqFunc{
	{
		Name: "uuid",
		Func: func(_ any, _ []any) any {
			return uuid.New().String()
		},
	},
}

// The return value could be any value depending upon how it's parsed
func EvaluateString(str string, input any, state *State, evaluationWrapper ...ExpressionWrapperFunc) (any, error) {
	// Check if the string is a runtime expression (e.g., ${ .some.path })
	if model.IsStrictExpr(str) {
		// Wrapper exists to allow JQ evaluation to be put inside a workflow to make deterministic
		fn := buildEvaluationWrapperFn(evaluationWrapper...)

		return fn(func() (any, error) {
			return evaluateJQExpression(model.SanitizeExpr(str), input, state)
		})
	}
	return str, nil
}

func buildEvaluationWrapperFn(evaluationWrapper ...ExpressionWrapperFunc) ExpressionWrapperFunc {
	var wrapperFn ExpressionWrapperFunc = func(f func() (any, error)) (any, error) {
		return f()
	}
	if len(evaluationWrapper) > 0 {
		// If a function is passed in, use that instead
		wrapperFn = evaluationWrapper[0]
	}

	return wrapperFn
}

func TraverseAndEvaluateObj(
	runtimeExpr *model.ObjectOrRuntimeExpr,
	input any,
	state *State,
	evaluationWrapper ...ExpressionWrapperFunc,
) (map[string]any, error) {
	if runtimeExpr == nil {
		return map[string]any{}, nil
	}

	// Default to a simple pass-thru function
	wrapperFn := buildEvaluationWrapperFn(evaluationWrapper...)

	s, err := traverseAndEvaluate(runtimeExpr.AsStringOrMap(), input, state, wrapperFn)
	if err != nil {
		return nil, err
	}

	if v, isMap := s.(map[string]any); isMap {
		return v, nil
	} else {
		return nil, fmt.Errorf("unknown data type")
	}
}

func traverseAndEvaluate(node, input any, state *State, evaluationWrapper ExpressionWrapperFunc) (any, error) {
	switch v := node.(type) {
	case map[string]any:
		// Traverse a object
		for key, value := range v {
			evaluatedValue, err := traverseAndEvaluate(value, input, state, evaluationWrapper)
			if err != nil {
				return nil, err
			}
			v[key] = evaluatedValue
		}
		return v, nil
	case []any:
		// Traverse an array
		for i, value := range v {
			evaluatedValue, err := traverseAndEvaluate(value, input, state, evaluationWrapper)
			if err != nil {
				return nil, err
			}
			v[i] = evaluatedValue
		}
		return v, nil
	case string:
		return EvaluateString(v, input, state, evaluationWrapper)
	default:
		// Return as-is
		return v, nil
	}
}

func evaluateJQExpression(expression string, input any, state *State) (any, error) {
	query, err := gojq.Parse(expression)
	if err != nil {
		return nil, fmt.Errorf("failed to parse jq expression: %s, error: %w", expression, err)
	}

	// Get the variable names & values in a single pass:
	names, values := getVariableNamesAndValues(state.GetAsMap())

	fns := []gojq.CompilerOption{
		gojq.WithVariables(names),
	}
	for _, j := range jqFuncs {
		fns = append(fns, gojq.WithFunction(j.Name, j.MinArgs, j.MaxArgs, j.Func))
	}

	code, err := gojq.Compile(query, fns...)
	if err != nil {
		return nil, fmt.Errorf("failed to compile jq expression: %s, error: %w", expression, err)
	}

	iter := code.Run(input, values...)
	result, ok := iter.Next()
	if !ok {
		return nil, fmt.Errorf("no result from jq evaluation")
	}

	// If there's an error from the jq engine, report it
	if errVal, isErr := result.(error); isErr {
		return nil, fmt.Errorf("jq evaluation error: %w", errVal)
	}

	return result, nil
}

func getVariableNamesAndValues(vars map[string]any) ([]string, []any) {
	names := make([]string, 0, len(vars))
	values := make([]any, 0, len(vars))

	for k, v := range vars {
		names = append(names, k)
		values = append(values, v)
	}
	return names, values
}

// func evaluateJQExpression(expression string, state *State) (any, error) {
// 	query, err := gojq.Parse(expression)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to parse jq expression: %s, error: %w", expression, err)
// 	}

// 	fns := make([]gojq.CompilerOption, 0)
// 	for _, j := range jqFuncs {
// 		fns = append(fns, gojq.WithFunction(j.Name, j.MinArgs, j.MaxArgs, j.Func))
// 	}

// 	code, err := gojq.Compile(query, fns...)
// 	if err != nil {
// 		return nil, fmt.Errorf("error compiling gojq code: %w", err)
// 	}

// 	iter := code.Run(state.GetAsMap())
// 	v, ok := iter.Next()
// 	if !ok {
// 		return nil, fmt.Errorf("no result from jq evaluation")
// 	}
// 	if errVal, isErr := v.(error); isErr {
// 		return nil, fmt.Errorf("jq evaluation error: %w", errVal)
// 	}

// 	return v, nil
// }
