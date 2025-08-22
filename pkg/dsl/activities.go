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
	"context"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
)

func GetActivityVars(ctx context.Context) HTTPData {
	info := activity.GetInfo(ctx)

	// Prepend with "_ta_" to avoid clashes
	v := HTTPData{
		"_ta_activity_id":               info.ActivityID,
		"_ta_activity_type_name":        info.ActivityType.Name,
		"_ta_attempt":                   info.Attempt,
		"_ta_deadline":                  info.Deadline,
		"_ta_heartbeat_token":           info.HeartbeatTimeout,
		"_ta_is_local_activity":         info.IsLocalActivity,
		"_ta_priority_key":              info.Priority.PriorityKey,
		"_ta_schedule_to_close_timeout": info.ScheduleToCloseTimeout,
		"_ta_scheduled_time":            info.ScheduledTime,
		"_ta_start_to_close_timeout":    info.StartToCloseTimeout,
		"_ta_started_time":              info.StartedTime,
		"_ta_task_queue":                info.TaskQueue,
		"_ta_task_token":                string(info.TaskToken),
		"_ta_workflow_namespace":        info.WorkflowNamespace,
		"_ta_workflow_execution_id":     info.WorkflowExecution.ID,
		"_ta_workflow_execution_run_id": info.WorkflowExecution.RunID,
	}

	if w := info.WorkflowType; w != nil {
		v["_ta_workflow_type_name"] = w.Name
	}

	return v
}

func GetWorkflowInfo(ctx workflow.Context) HTTPData {
	info := workflow.GetInfo(ctx)

	// Prepend with "_tw_" to avoid clashes
	v := HTTPData{
		"_tw_attempt":                    info.Attempt,
		"_tw_binary_checksum":            info.BinaryChecksum,
		"_tw_continued_execution_run_id": info.ContinuedExecutionRunID,
		"_tw_cron_schedule":              info.CronSchedule,
		"_tw_first_run_id":               info.FirstRunID,
		"_tw_namespace":                  info.Namespace,
		"_tw_original_run_id":            info.OriginalRunID,
		"_tw_parent_workflow_namespace":  info.ParentWorkflowNamespace,
		"_tw_priority_key":               info.Priority.PriorityKey,
		"_tw_task_queue_name":            info.TaskQueueName,
		"_tw_workflow_execution_id":      info.WorkflowExecution.ID,
		"_tw_workflow_execution_run_id":  info.WorkflowExecution.RunID,
		"_tw_workflow_execution_timeout": info.WorkflowExecutionTimeout,
		"_tw_workflow_start_time":        info.WorkflowStartTime,
		"_tw_workflow_type_name":         info.WorkflowType.Name,
	}

	if r := info.RootWorkflowExecution; r != nil {
		v["_tw_root_workflow_execution_id"] = r.ID
		v["_tw_root_workflow_execution_run_id"] = r.RunID
	}

	if p := info.ParentWorkflowExecution; p != nil {
		v["_tw_parent_workflow_execution_id"] = p.ID
		v["_tw_parent_workflow_execution_run_id"] = p.RunID
	}

	return v
}
