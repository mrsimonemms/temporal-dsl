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

package fork

import (
	_ "embed"

	"github.com/mrsimonemms/temporal-dsl/tests/e2e/utils"
)

var testCase = utils.TestCase{
	Name:         "fork",
	WorkflowPath: "workflow.yaml",
	ExpectedOutput: map[string]any{
		"fork": map[string]any{
			"callHTTP1": map[string]any{
				"id":    "1",
				"title": "a title",
				"views": float64(100),
			},
			"callHTTP2": map[string]any{
				"id":    "2",
				"title": "another title",
				"views": float64(200),
			},
		},
	},
	Test: utils.RunToCompletion,
}

func init() {
	utils.AddTestCase(testCase)
}
