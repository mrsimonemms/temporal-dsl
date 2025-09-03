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

package dsl

import (
	"fmt"
	"strconv"

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"go.temporal.io/sdk/workflow"
)

// Wrap all set values in a SideEffect to allow for generated values
// to be safely used. This avoid non-deterministic errors, which are a
// pain in the arse in Temporalland
func setTaskValue(ctx workflow.Context, input string, data *Variables) (string, error) {
	logger := workflow.GetLogger(ctx)
	var str string
	err := workflow.SideEffect(ctx, func(ctx workflow.Context) any {
		return MustParseVariables(input, data)
	}).Get(&str)
	if err != nil {
		logger.Error("Unable to generate side effect value", "error", err)
		return "", fmt.Errorf("unable to generate side effect value: %w", err)
	}

	return str, nil
}

func setTaskInterpolate(ctx workflow.Context, keyID, input any, data *Variables) (outputValue any, err error) {
	logger := workflow.GetLogger(ctx)

	switch v := input.(type) {
	case map[string]any:
		logger.Debug("Parsing as JSON object", "key", keyID)
		// Create a new object
		obj := make(map[string]any)

		// Iterate over each item
		for i, item := range v {
			// Interpolate the object key
			var key any
			var keyStr string
			key, err = setTaskInterpolate(ctx, i, i, data)
			if err != nil {
				return outputValue, err
			}
			if k, ok := key.(string); ok {
				keyStr = k
			} else {
				err = fmt.Errorf("%w: must be %s", ErrInvalidType, "string")
				return outputValue, err
			}

			var o any
			o, err = setTaskInterpolate(ctx, i, item, data)
			if err != nil {
				return outputValue, err
			}

			obj[keyStr] = o
		}
		outputValue = obj
	case []any:
		logger.Debug("Parsing as JSON array", "key", keyID)
		// Create a new array
		arr := make([]any, 0)

		// Iterate over each item
		for i, item := range v {
			var o any
			o, err = setTaskInterpolate(ctx, strconv.Itoa(i), item, data)
			if err != nil {
				return outputValue, err
			}

			arr = append(arr, o)
		}
		outputValue = arr
	case string:
		logger.Debug("Parsing as JSON string", "key", keyID)
		outputValue, err = setTaskValue(ctx, v, data)
	default:
		logger.Debug("Maintaining JSON type", "key", keyID)
		outputValue = v
	}
	if err != nil {
		return outputValue, err
	}

	return outputValue, err
}

func setTaskImpl(task *model.SetTask) TemporalWorkflowFunc {
	return func(ctx workflow.Context, data *Variables, output map[string]OutputType) (map[string]OutputType, error) {
		out := make(map[string]OutputType)
		for key, value := range task.Set {
			parsedValue, err := setTaskInterpolate(ctx, key, value, data)
			if err != nil {
				return nil, err
			}

			data.Data[key] = parsedValue
		}

		return out, nil
	}
}
