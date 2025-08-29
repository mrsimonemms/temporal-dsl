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

	"github.com/mrsimonemms/golang-helpers/temporal"
	"github.com/mrsimonemms/temporal-dsl/pkg/dsl"
	"github.com/rs/zerolog/log"
	"go.temporal.io/sdk/client"
)

func main() {
	// The client is a heavyweight object that should be created once per process.
	c, err := temporal.NewConnection(
		temporal.WithHostPort(os.Getenv("TEMPORAL_ADDRESS")),
		temporal.WithNamespace(os.Getenv("TEMPORAL_NAMESPACE")),
		temporal.WithAPICredentials(os.Getenv("TEMPORAL_API_KEY")),
		temporal.WithTLS(os.Getenv("TEMPORAL_TLS") == "true"),
		temporal.WithZerolog(&log.Logger),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to create client")
	}
	defer c.Close()

	workflowOptions := client.StartWorkflowOptions{
		TaskQueue: "temporal-dsl",
	}

	ctx := context.Background()
	orderTypes := []string{
		"electronic",
		"physical",
		"unknown",
	}

	for _, orderType := range orderTypes {
		log.Info().Str("orderType", orderType).Msg("Triggering new order")
		we, err := c.ExecuteWorkflow(ctx, workflowOptions, "switch", dsl.HTTPData{
			"orderType": orderType,
		})
		if err != nil {
			//nolint:gocritic
			log.Fatal().Err(err).Msg("Error executing workflow")
		}

		log.Info().Str("workflowId", we.GetID()).Str("runId", we.GetRunID()).Msg("Started workflow")

		var result map[string]dsl.OutputType
		if err := we.Get(ctx, &result); err != nil {
			log.Fatal().Err(err).Msg("Error getting response")
		}
	}
}
