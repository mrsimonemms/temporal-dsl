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
	"os"
	"time"

	gh "github.com/mrsimonemms/golang-helpers"
	"github.com/mrsimonemms/golang-helpers/temporal"
	"github.com/mrsimonemms/temporal-dsl/pkg/dsl"
	"github.com/rs/zerolog/log"
	"go.temporal.io/sdk/client"
)

func exec() error {
	// The client is a heavyweight object that should be created once per process.
	c, err := temporal.NewConnection(
		temporal.WithHostPort(os.Getenv("TEMPORAL_ADDRESS")),
		temporal.WithNamespace(os.Getenv("TEMPORAL_NAMESPACE")),
		temporal.WithAPICredentials(os.Getenv("TEMPORAL_API_KEY")),
		temporal.WithTLS(os.Getenv("TEMPORAL_TLS") == "true"),
		temporal.WithZerolog(&log.Logger),
	)
	if err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   "Unable to create client",
		}
	}
	defer c.Close()

	workflowOptions := client.StartWorkflowOptions{
		TaskQueue: "temporal-dsl",
	}

	ctx := context.Background()
	we, err := c.ExecuteWorkflow(ctx, workflowOptions, "listen", dsl.HTTPData{
		"userId": 3,
	})
	if err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   "Error executing workflow",
		}
	}

	log.Info().Str("workflowId", we.GetID()).Str("runId", we.GetRunID()).Msg("Started workflow")

	time.Sleep(time.Second * 2)

	log.Info().Str("event", "event1").Msg("Triggering update")
	updateHandle1, err := c.UpdateWorkflow(ctx, client.UpdateWorkflowOptions{
		WorkflowID:   we.GetID(),
		WaitForStage: client.WorkflowUpdateStageCompleted,
		UpdateName:   "com.fake-hospital.vitals.measurements.temperature",
		Args: []any{
			dsl.HTTPData{
				"temperature": 39,
			},
		},
	})
	if err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   "Error updating",
		}
	}

	if err := updateHandle1.Get(ctx, nil); err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   "Update failed",
		}
	}

	time.Sleep(time.Second * 2)

	log.Info().Str("event", "event2").Msg("Triggering update")
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
		return gh.FatalError{
			Cause: err,
			Msg:   "Error updating",
		}
	}

	if err := updateHandle2.Get(ctx, nil); err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   "Update failed",
		}
	}
	return nil
}

func main() {
	if err := exec(); err != nil {
		os.Exit(gh.HandleFatalError(err))
	}
}
