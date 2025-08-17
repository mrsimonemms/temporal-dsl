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

	"github.com/google/uuid"
	"github.com/mrsimonemms/golang-helpers/temporal"
	"github.com/rs/zerolog/log"
	"go.temporal.io/sdk/client"
)

type State struct {
	ID       uuid.UUID `json:"id"`
	Progress int       `json:"progressPercentage"`
	Status   string    `json:"status"`
}

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
		TaskQueue: "temporal-dsl",
	}

	ctx := context.Background()
	we, err := c.ExecuteWorkflow(ctx, workflowOptions, "query")
	if err != nil {
		//nolint:gocritic
		log.Fatal().Err(err).Msg("Error executing workflow")
	}

	log.Info().Str("workflowId", we.GetID()).Str("runId", we.GetRunID()).Msg("Started workflow")

	quit := make(chan bool)
	go func() {
		for {
			select {
			case <-quit:
				log.Info().Msg("Workflow completed")
				return
			default:
				res, err := c.QueryWorkflow(ctx, we.GetID(), "", "get_state")
				if err != nil {
					log.Fatal().Err(err).Msg("Error querying workflow")
				}

				var state State
				if err := res.Get(&state); err != nil {
					log.Fatal().Err(err).Msg("Error getting query result")
				}
				log.Info().Interface("Query result", state).Msg("Response from query")

				time.Sleep(time.Second * 2)
			}
		}
	}()

	if err := we.Get(ctx, nil); err != nil {
		log.Fatal().Err(err).Msg("Error getting response")
	}

	quit <- true
}
