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

	"github.com/serverlessworkflow/sdk-go/v3/impl/expr"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	goPanic                 = "https://go.dev/panic"
	temporaErrlNonRetryable = "https://temporal.io/errors/nonretryable"
)

// Serverless Workflow native errors
var raiseErrFuncMapping = map[string]func(error, string) *model.Error{
	model.ErrorTypeAuthentication: model.NewErrAuthentication,
	model.ErrorTypeValidation:     model.NewErrValidation,
	model.ErrorTypeCommunication:  model.NewErrCommunication,
	model.ErrorTypeAuthorization:  model.NewErrAuthorization,
	model.ErrorTypeConfiguration:  model.NewErrConfiguration,
	model.ErrorTypeExpression:     model.NewErrExpression,
	model.ErrorTypeRuntime:        model.NewErrRuntime,
	model.ErrorTypeTimeout:        model.NewErrTimeout,
}

var temporalErrMapping = map[string]func(error, string) error{
	// Special Temporal types
	goPanic: func(_ error, msg string) error {
		panic(msg)
	},
	temporaErrlNonRetryable: func(err error, msg string) error {
		return temporal.NewNonRetryableApplicationError(msg, temporaErrlNonRetryable, err)
	},
}

func raiseTaskImpl(task *model.RaiseTask, taskName string) TemporalWorkflowFunc {
	return func(ctx workflow.Context, data *Variables, output map[string]OutputType) (outputData map[string]OutputType, err error) {
		logger := workflow.GetLogger(ctx)
		logger.Debug("Raising error")

		gtx := context.Background()

		info := workflow.GetInfo(ctx)
		instanceID := info.WorkflowExecution.ID

		var raiseErr *model.Error
		var titleResult any = ""
		var detailResult any = ""
		if definition := task.Raise.Error.Definition; definition != nil {
			if detail := definition.Title; detail != nil {
				detailResult, err = expr.TraverseAndEvaluateObj(
					detail.AsObjectOrRuntimeExpr(),
					data,
					taskName,
					gtx,
				)
				if err != nil {
					logger.Error("Error finding error definition", "error", err)
					err = fmt.Errorf("error finding error definition: %w", err)
					return nil, err
				}
			}

			if title := definition.Title; title != nil {
				titleResult, err = expr.TraverseAndEvaluateObj(
					task.Raise.Error.Definition.Title.AsObjectOrRuntimeExpr(),
					data,
					taskName,
					gtx,
				)
				if err != nil {
					logger.Error("Error finding error title definition", "error", err)
					err = fmt.Errorf("error finding error title definition: %w", err)
					return nil, err
				}
			}

			if raiseErrF, ok := raiseErrFuncMapping[definition.Type.String()]; ok {
				raiseErr = raiseErrF(fmt.Errorf("%v", detailResult), instanceID)
			} else if temporalErrF, ok := temporalErrMapping[definition.Title.String()]; ok {
				return nil, temporalErrF(fmt.Errorf("%v", detailResult), instanceID)
			} else {
				raiseErr = definition
				raiseErr.Detail = model.NewStringOrRuntimeExpr(fmt.Sprintf("%v", detailResult))
				raiseErr.Instance = &model.JsonPointerOrRuntimeExpression{
					Value: instanceID,
				}
			}

			raiseErr.Title = model.NewStringOrRuntimeExpr(fmt.Sprintf("%v", titleResult))
			raiseErr.Status = definition.Status
		}

		return nil, raiseErr
	}
}
