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

package workflow

import (
	"context"
	"fmt"

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"go.temporal.io/sdk/client"
)

const scheduleID = "trigger_payments"

func UpsertSchedule(ctx context.Context, temporalClient client.Client, workflow *Workflow) error {
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

	// Build Temporal schedules
	scheduleSpec := buildTemporalScheduleSpec(*schedule)

	// Convert the Serverless Workflow schedule to a Temporal schedule
	opts := client.ScheduleOptions{
		ID:   scheduleID,
		Spec: scheduleSpec,
		Action: &client.ScheduleWorkflowAction{
			Workflow:  "tits",
			TaskQueue: "arse",
		},
	}

	if _, err := scheduleClient.Create(ctx, opts); err != nil {
		return fmt.Errorf("error creating schedule: %w", err)
	}

	return nil
}

// Converts the Serverless Workflow schedule to Temporal schedule spec
func buildTemporalScheduleSpec(schedule model.Schedule) client.ScheduleSpec {
	calendars := make([]client.ScheduleCalendarSpec, 0)
	cronExpression := make([]string, 0)
	intervals := make([]client.ScheduleIntervalSpec, 0)

	if schedule.Cron != "" {
		cronExpression = append(cronExpression, schedule.Cron)
	}
	if schedule.Every != nil {
		cal := client.ScheduleCalendarSpec{}

		if duration := schedule.Every.AsInline(); duration != nil {
			cal.Hour = []client.ScheduleRange{
				{
					Start: int(duration.Hours + (duration.Days * 24)),
				},
			}
			cal.Minute = []client.ScheduleRange{
				{
					Start: int(duration.Minutes),
				},
			}
			fmt.Println(schedule.Every.AsInline().Days)

			// Days         int32 `json:"days,omitempty"`
			// Hours        int32 `json:"hours,omitempty"`
			// Minutes      int32 `json:"minutes,omitempty"`
			// Seconds      int32 `json:"seconds,omitempty"`
			// Milliseconds int32 `json:"milliseconds,omitempty"`

			calendars = append(calendars, cal)
		}
	}

	return client.ScheduleSpec{
		Calendars:       calendars,
		CronExpressions: cronExpression,
		Intervals:       intervals,
	}
}
