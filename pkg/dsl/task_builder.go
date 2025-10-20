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
	"fmt"

	"github.com/mrsimonemms/temporal-dsl/pkg/dsl/tasks"
	"github.com/rs/zerolog/log"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

func NewWorkflow(temporalWorker worker.Worker, doc *model.Workflow) error {
	workflowName := doc.Document.Name
	l := log.With().Str("workflowName", workflowName).Logger()

	l.Debug().Msg("Creating new Do builder")
	doBuilder, err := tasks.NewDoTaskBuilder(
		temporalWorker,
		&model.DoTask{Do: doc.Do},
		workflowName,
		doc,
		// Disable registering if it's the prime workflow
		tasks.DoTaskOpts{DisableRegisterWorkflow: true},
	)
	if err != nil {
		l.Error().Err(err).Msg("Error creating Do builder")
		return fmt.Errorf("error creating do builder: %w", err)
	}

	l.Debug().Msg("Building workflow")
	wf, err := doBuilder.Build()
	if err != nil {
		l.Error().Err(err).Msg("Error building workflow")
		return fmt.Errorf("error building workflow: %w", err)
	}

	// Wrap the function as the prime function
	var workflowFn tasks.TemporalWorkflowFunc = func(ctx workflow.Context, input any, state map[string]any) (output any, err error) {
		logger := workflow.GetLogger(ctx)
		logger.Info("Starting workflow")

		if state == nil {
			logger.Debug("Creating new empty state map")
			state = map[string]any{}
		}

		return wf(ctx, input, state)
	}

	l.Debug().Msg("Registering workflow")
	temporalWorker.RegisterWorkflowWithOptions(workflowFn, workflow.RegisterOptions{
		Name: workflowName,
	})

	for _, a := range tasks.ActivitiesList() {
		l.Debug().Msg("Registering activity")
		temporalWorker.RegisterActivity(a)
	}

	return nil
}
