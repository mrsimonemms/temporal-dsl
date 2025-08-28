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
	"fmt"

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"go.temporal.io/sdk/client"
)

const (
	scheduleMetadataScheduleID       string = "scheduleId"
	scheduleMetadataScheduleWorkflow string = "scheduleWorkflowName"
	scheduleMetadataInput            string = "scheduleInput"
)

func UpsertSchedule(ctx context.Context, temporalClient client.Client, workflow *Workflow, taskQueue string) error {
	// Based the schedule ID on the workflow name
	scheduleID := fmt.Sprintf("dsl_%s", workflow.WorkflowName())
	if s, ok := workflow.Document().Metadata[scheduleMetadataScheduleID]; ok {
		if sID, ok := s.(string); ok {
			// Schedule ID is set in the metadata
			scheduleID = sID
		} else {
			fmt.Println(sID)
			return fmt.Errorf("schedule id must be a string")
		}
	}

	schedule := workflow.Schedule()
	scheduleClient := temporalClient.ScheduleClient()

	// Always delete matching schedules
	schedules, err := scheduleClient.List(ctx, client.ScheduleListOptions{})
	if err != nil {
		return fmt.Errorf("error listing temporal schedules: %w", err)
	}

	for schedules.HasNext() {
		s, err := schedules.Next()
		if err != nil {
			return fmt.Errorf("unable to get schedule: %w", err)
		}

		// Find and destroy the schedule
		if s.ID == scheduleID {
			handler := scheduleClient.GetHandle(ctx, s.ID)

			if err := handler.Delete(ctx); err != nil {
				return fmt.Errorf("error deleting workflow schedule: %w", err)
			}
		}
	}

	// If no schedule set, nothing to do now
	if schedule == nil {
		return nil
	}

	var workflowName string
	if t, ok := workflow.Document().Metadata[scheduleMetadataScheduleWorkflow]; ok {
		if t, ok := t.(string); ok {
			workflowName = t
		} else {
			return fmt.Errorf("schedule workflow name must be a string")
		}
	} else {
		return ErrScheduleNoWorkflowName
	}

	// Build Temporal schedules
	scheduleSpec, err := buildTemporalScheduleSpec(*schedule)
	if err != nil {
		return fmt.Errorf("error converting schedule to temporal: %w", err)
	}

	var input []any
	if in, ok := workflow.Document().Metadata[scheduleMetadataInput]; ok {
		if i, ok := in.([]any); ok {
			input = i
		} else {
			return fmt.Errorf("schedule input must be in array format")
		}
	}

	// Convert the Serverless Workflow schedule to a Temporal schedule
	opts := client.ScheduleOptions{
		ID:   scheduleID,
		Spec: *scheduleSpec,
		Action: &client.ScheduleWorkflowAction{
			Workflow:  workflowName,
			TaskQueue: taskQueue,
			Args:      input,
		},
	}

	if _, err := scheduleClient.Create(ctx, opts); err != nil {
		return fmt.Errorf("error creating schedule: %w", err)
	}

	return nil
}

// Converts the Serverless Workflow schedule to Temporal schedule spec
func buildTemporalScheduleSpec(schedule model.Schedule) (*client.ScheduleSpec, error) {
	calendars := make([]client.ScheduleCalendarSpec, 0)
	cronExpression := make([]string, 0)
	intervals := make([]client.ScheduleIntervalSpec, 0)

	if schedule.Cron != "" {
		cronExpression = append(cronExpression, schedule.Cron)
	}
	if schedule.Every != nil {
		if duration := schedule.Every; duration != nil {
			intervals = append(intervals, client.ScheduleIntervalSpec{
				Every: ToDuration(duration),
			})
		}
	}
	if schedule.After != nil {
		return nil, fmt.Errorf("schedule.after not supported")
	}

	return &client.ScheduleSpec{
		Calendars:       calendars,
		CronExpressions: cronExpression,
		Intervals:       intervals,
	}, nil
}
