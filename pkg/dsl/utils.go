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
	"bytes"
	"fmt"
	"maps"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/itchyny/gojq"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"go.temporal.io/sdk/temporal"
	"gopkg.in/yaml.v3"
)

func CheckIfStatement(ifStatement *model.RuntimeExpression, input *Variables) (toRun bool, err error) {
	if ifStatement != nil {
		var query *gojq.Query

		expression := model.SanitizeExpr(ifStatement.String())
		query, err = gojq.Parse(expression)
		if err != nil {
			err = fmt.Errorf("unable to parse if statement as expression: %w", err)
			return toRun, err
		}

		// For some reason, GoJQ doesn't like HTTPData even though it's map[string]any 😕
		data := make(map[string]any)
		maps.Copy(data, input.Data)

		iter := query.Run(data)
		for {
			v, ok := iter.Next()
			if !ok {
				break
			}
			if err, ok = v.(error); ok {
				// Any JQ error will be considered a non-retryable error
				err = temporal.NewNonRetryableApplicationError("Error parsing if statement in JQ", string(IfStatementErr), err)
				return toRun, err
			}

			switch r := v.(type) {
			case bool:
				toRun = r
			case string:
				// Can resolve "TRUE" or "1"
				toRun = strings.EqualFold(r, "TRUE") || r == "1"
			}
		}
	} else {
		// No statement - continue with true
		toRun = true
	}

	return toRun, err
}

func GenerateChildWorkflowName(prefix string, prefixes ...string) string {
	prefixes = append([]string{prefix}, prefixes...)

	return fmt.Sprintf("workflow_%s", strings.Join(prefixes, "_"))
}

// Interpolate the given input. Unlike the interpolation in the SetTask, this
// only works with the given data and should be used for getting data rather
// than setting data - this may given non-deterministic errors
func Interpolate(input any, data *Variables) (outputValue any, err error) {
	switch v := input.(type) {
	case map[string]any:
		// Create a new object
		obj := make(map[string]any)

		// Iterate over each item
		for i, item := range v {
			// Interpolate the object key
			var key any
			var keyStr string
			key, err = Interpolate(i, data)
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
			o, err = Interpolate(item, data)
			if err != nil {
				return outputValue, err
			}

			obj[keyStr] = o
		}
		outputValue = obj
	case []any:
		// Create a new array
		arr := make([]any, 0)

		// Iterate over each item
		for _, item := range v {
			var o any
			o, err = Interpolate(item, data)
			if err != nil {
				return outputValue, err
			}

			arr = append(arr, o)
		}
		outputValue = arr
	case string:
		outputValue, err = ParseVariables(v, data)
	default:
		outputValue = v
	}

	return outputValue, err
}

// Parses a string with variables
func ParseVariables(input string, data *Variables) (string, error) {
	if data == nil {
		return input, nil
	}

	t, err := template.New("values").
		Funcs(sprig.FuncMap()).
		Parse(input)
	if err != nil {
		return "", fmt.Errorf("error creating template instance: %w", err)
	}

	buf := new(bytes.Buffer)
	if err := t.Execute(buf, data.Data); err != nil {
		return "", fmt.Errorf("error executing template: %w", err)
	}

	return buf.String(), nil
}

func MustParseVariables(input string, data *Variables) string {
	str, err := ParseVariables(input, data)
	if err != nil {
		panic(err)
	}

	return str
}

func SlicesEqual[T comparable](s []T, v T) bool {
	for _, r := range s {
		if r != v {
			return false
		}
	}
	return true
}

func FromYAML(input any) (*HTTPData, error) {
	if i, ok := input.(string); ok {
		var data *HTTPData
		if err := yaml.Unmarshal([]byte(i), &data); err != nil {
			return nil, fmt.Errorf("error converting json: %w", err)
		}
		return data, nil
	}

	return nil, ErrNotString
}

// Converts the SW duration to a time Duration
func ToDuration(v *model.Duration) time.Duration {
	inline := v.AsInline()

	var duration time.Duration
	duration += time.Millisecond * time.Duration(inline.Milliseconds)
	duration += time.Second * time.Duration(inline.Seconds)
	duration += time.Minute * time.Duration(inline.Minutes)
	duration += time.Hour * time.Duration(inline.Hours)
	duration += (time.Hour * 24) * time.Duration(inline.Days)

	return duration
}

// Functionally equivalent to JS's Array.prorotype.every function
// @link https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Array/every
func SliceEvery[T any](items []T, predicate func(T) bool) bool {
	for _, v := range items {
		if !predicate(v) {
			return false
		}
	}
	return true
}
