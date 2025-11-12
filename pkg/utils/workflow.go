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
	"strings"

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"go.temporal.io/sdk/temporal"
)

func CheckIfStatement(ifStatement *model.RuntimeExpression, state *State) (bool, error) {
	if ifStatement == nil {
		return true, nil
	}

	fmt.Printf("%+v\n", state.Data)
	// fmt.Printf("%+v\n", state.Context)
	fmt.Println(ifStatement.String())

	res, err := EvaluateString(ifStatement.String(), nil, state)
	if err != nil {
		// Treat a parsing error as non-retryable
		return false, temporal.NewNonRetryableApplicationError("Error parsing if statement", "If statement error", err)
	}
	fmt.Println(res)

	// Response can be a boolean, "TRUE" (case-insensitive) or "1"
	switch r := res.(type) {
	case bool:
		return r, nil
	case string:
		return strings.EqualFold(r, "TRUE") || r == "1", nil
	default:
		return false, temporal.NewNonRetryableApplicationError(
			"If statement response type unknown",
			"If statement error",
			fmt.Errorf("response not string or bool"),
		)
	}
}

func GenerateChildWorkflowName(prefix string, prefixes ...string) string {
	prefixes = append([]string{prefix}, prefixes...)

	return fmt.Sprintf("workflow_%s", strings.Join(prefixes, "_"))
}
