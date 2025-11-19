package main

import (
	"context"
	"fmt"
	"os"

	gh "github.com/mrsimonemms/golang-helpers"
	"github.com/mrsimonemms/golang-helpers/temporal"
	"github.com/rs/zerolog/log"
	"go.temporal.io/sdk/client"
)

func exec() error {
	c, err := temporal.NewConnectionWithEnvvars(
		temporal.WithZerolog(&log.Logger),
	)
	if err != nil {
		return gh.FatalError{Cause: err, Msg: "Unable to create client"}
	}
	defer c.Close()

	workflowOptions := client.StartWorkflowOptions{TaskQueue: "temporal-dsl"}

	input := map[string]any{
		"userId": "dsl-user-42",
	}

	ctx := context.Background()
	we, err := c.ExecuteWorkflow(ctx, workflowOptions, "activity-call", input)
	if err != nil {
		return gh.FatalError{Cause: err, Msg: "Error executing workflow"}
	}

	log.Info().Str("workflowId", we.GetID()).Str("runId", we.GetRunID()).Msg("Started workflow")

	var result map[string]any
	if err := we.Get(ctx, &result); err != nil {
		return gh.FatalError{Cause: err, Msg: "Error getting response"}
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
