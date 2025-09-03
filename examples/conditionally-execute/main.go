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
	"fmt"
	"math/rand/v2"
	"os"

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

	//nolint:gosec
	i := rand.IntN(10)
	log.Info().Int("randomInteger", i).Msg("Triggering workflow")

	ctx := context.Background()
	we, err := c.ExecuteWorkflow(ctx, workflowOptions, "conditional", dsl.HTTPData{
		"randomInteger": i,
	})
	if err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   "Error executing workflow",
		}
	}

	log.Info().Str("workflowId", we.GetID()).Str("runId", we.GetRunID()).Msg("Started workflow")

	var result map[string]dsl.OutputType
	if err := we.Get(ctx, &result); err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   "Error getting response",
		}
	}

	log.Info().Interface("result", result).Msg("Workflow completed")

	fmt.Println("===")
	fmt.Printf("%+v\n", result)
	fmt.Println("===")

	return nil
}

func main() {
	if err := exec(); err != nil {
		os.Exit(gh.HandleFatalError(err))
	}
}
