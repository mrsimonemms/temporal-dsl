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

package main

import (
	"context"
	"time"

	"github.com/mrsimonemms/golang-helpers/temporal"
	"github.com/mrsimonemms/temporal-dsl/pkg/workflow"
	"github.com/rs/zerolog/log"
	"go.temporal.io/sdk/client"
)

func main() {
	// The client is a heavyweight object that should be created once per process.
	c, err := client.Dial(client.Options{
		Logger: temporal.NewZerologHandler(&log.Logger),
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to create client")
	}
	defer c.Close()

	workflowOptions := client.StartWorkflowOptions{
		TaskQueue: "serverless-workflow",
	}

	ctx := context.Background()
	we, err := c.ExecuteWorkflow(ctx, workflowOptions, "listen", workflow.HTTPData{
		"userId": 3,
	})
	if err != nil {
		//nolint:gocritic
		log.Fatal().Err(err).Msg("Error executing workflow")
	}

	log.Info().Str("workflowId", we.GetID()).Str("runId", we.GetRunID()).Msg("Started workflow")

	time.Sleep(time.Second * 2)

	log.Info().Str("event", "eve").Msg("Triggering update")
	updateHandle1, err := c.UpdateWorkflow(ctx, client.UpdateWorkflowOptions{
		WorkflowID:   we.GetID(),
		WaitForStage: client.WorkflowUpdateStageCompleted,
		UpdateName:   "com.fake-hospital.vitals.measurements.temperature",
		Args: []any{
			workflow.HTTPData{
				"temperature": 39,
			},
		},
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Error updating")
	}

	if err := updateHandle1.Get(ctx, nil); err != nil {
		log.Fatal().Err(err).Msg("Update failed")
	}

	time.Sleep(time.Second * 2)

	log.Info().Str("event", "eve").Msg("Triggering update")
	updateHandle2, err := c.UpdateWorkflow(ctx, client.UpdateWorkflowOptions{
		WorkflowID:   we.GetID(),
		WaitForStage: client.WorkflowUpdateStageCompleted,
		UpdateName:   "com.fake-hospital.vitals.measurements.bpm",
		Args: []any{
			map[string]any{
				"bpm": 130,
			},
		},
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Error updating")
	}

	if err := updateHandle2.Get(ctx, nil); err != nil {
		log.Fatal().Err(err).Msg("Update failed")
	}
}
