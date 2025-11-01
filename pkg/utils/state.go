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

package utils

import (
	"maps"
	"strings"

	"github.com/rs/zerolog/log"
	swUtils "github.com/serverlessworkflow/sdk-go/v3/impl/utils"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"go.temporal.io/sdk/workflow"
)

type State struct {
	Data   map[string]any `json:"data"`            // Data stored along the way
	Env    map[string]any `json:"env"`             // Available environment variables
	Input  any            `json:"input,omitempty"` // The input given by the caller
	Output map[string]any `json:"output"`          // What will be output to the caller
}

func (s *State) init() *State {
	if s.Env == nil {
		s.Env = map[string]any{}
	}
	if s.Data == nil {
		s.Data = map[string]any{}
	}
	if s.Output == nil {
		s.Output = map[string]any{}
	}

	return s
}

func (s *State) AddData(data map[string]any) *State {
	maps.Copy(s.Data, data)

	return s
}

func (s *State) AddOutput(task model.Task, output any) *State {
	if output != nil {
		if export := task.GetBase().Export; export != nil {
			if exportAs := export.As; exportAs != nil {
				// Trim runtime expression wrapper
				key := strings.Trim(exportAs.String(), "{}")
				log.Debug().Any("key", key).Msg("Add task output to state")

				s.Output[key] = output
			}
		}
	}

	return s
}

func (s *State) AddWorkflowInfo(ctx workflow.Context) *State {
	info := workflow.GetInfo(ctx)

	workflowData := map[string]any{
		"attempt":                    info.Attempt,
		"binary_checksum":            info.BinaryChecksum,
		"continued_execution_run_id": info.ContinuedExecutionRunID,
		"cron_schedule":              info.CronSchedule,
		"first_run_id":               info.FirstRunID,
		"namespace":                  info.Namespace,
		"original_run_id":            info.OriginalRunID,
		"parent_workflow_namespace":  info.ParentWorkflowNamespace,
		"priority_key":               info.Priority.PriorityKey,
		"task_queue_name":            info.TaskQueueName,
		"workflow_execution_id":      info.WorkflowExecution.ID,
		"workflow_execution_run_id":  info.WorkflowExecution.RunID,
		"workflow_execution_timeout": info.WorkflowExecutionTimeout,
		"workflow_start_time":        info.WorkflowStartTime,
		"workflow_type_name":         info.WorkflowType.Name,
	}

	if r := info.RootWorkflowExecution; r != nil {
		workflowData["root_workflow_execution_id"] = r.ID
		workflowData["root_workflow_execution_run_id"] = r.RunID
	}

	if p := info.ParentWorkflowExecution; p != nil {
		workflowData["parent_workflow_execution_id"] = p.ID
		workflowData["parent_workflow_execution_run_id"] = p.RunID
	}

	s.AddData(map[string]any{
		"workflow": workflowData,
	})

	return s
}

func (s *State) ClearOutput() *State {
	s.Output = map[string]any{}
	return s
}

func (s *State) Clone() *State {
	s1 := NewState()

	s1.Data = swUtils.DeepClone(s.Data)
	s1.Env = swUtils.DeepClone(s.Env)
	s1.Input = swUtils.DeepCloneValue(s.Input)
	s1.Output = swUtils.DeepClone(s.Output)

	return s1
}

// Returns the state as a map. This can be used for
func (s *State) GetAsMap() map[string]any {
	s1 := s.Clone()

	return map[string]any{
		"data":   s1.Data,
		"env":    s1.Env,
		"input":  s1.Input,
		"output": s1.Output,
	}
}

func NewState() *State {
	s := &State{}
	return s.init()
}
