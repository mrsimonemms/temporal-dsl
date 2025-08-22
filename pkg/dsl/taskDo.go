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

	"github.com/serverlessworkflow/sdk-go/v3/model"
)

// A Do task configures a new workflow
func doTaskImpl(
	do *model.DoTask,
	task *model.TaskItem,
	workflowInst *Workflow,
) ([]*TemporalWorkflow, error) {
	// This doesn't implement the if statement as it
	// doesn't make sense to conditionally register a workflow
	temporalWorkflows, err := workflowInst.workflowBuilder(do.Do, task.Key)
	if err != nil {
		return nil, fmt.Errorf("error building additional do workflows: %w", err)
	}

	return temporalWorkflows, nil
}
