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

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	"github.com/serverlessworkflow/sdk-go/v3/model"
)

type ValidationErrors struct {
	Key     string
	Message string
}

// Validation of the schema is handled separately. This validates that there is
// nothing used we've not implemented. This should reduce over time.
func validateTaskSupported(task *model.TaskItem) error {
	if doTask := task.AsDoTask(); doTask != nil {
		// Do task - iterate through these
		for _, t := range *doTask.Do {
			if err := validateTaskSupported(t); err != nil {
				return err
			}
		}
	}

	if emit := task.AsEmitTask(); emit != nil {
		return fmt.Errorf("%w: emit", ErrUnsupportedTask)
	}
	if forTask := task.AsForTask(); forTask != nil {
		return fmt.Errorf("%w: for", ErrUnsupportedTask)
	}
	if grpc := task.AsCallGRPCTask(); grpc != nil {
		return fmt.Errorf("%w: grpc", ErrUnsupportedTask)
	}
	if openapi := task.AsCallOpenAPITask(); openapi != nil {
		return fmt.Errorf("%w: openapi", ErrUnsupportedTask)
	}
	if run := task.AsRunTask(); run != nil {
		// Only Workflow tasks are implemented
		if run.Run.Workflow == nil {
			return fmt.Errorf("%w: run", ErrUnsupportedTask)
		}
	}
	if try := task.AsTryTask(); try != nil {
		return fmt.Errorf("%w: try", ErrUnsupportedTask)
	}
	return nil
}

func (w *Workflow) Validate() ([]ValidationErrors, error) {
	enTrans := en.New()
	uni := ut.New(enTrans)
	trans, _ := uni.GetTranslator(enTrans.Locale())

	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := en_translations.RegisterDefaultTranslations(validate, trans); err != nil {
		return nil, fmt.Errorf("error registering validator translations: %w", err)
	}

	// Combine validation errors
	var vErrs []ValidationErrors

	for _, task := range *w.wf.Do {
		if err := validateTaskSupported(task); err != nil {
			return nil, err
		}

		if len(task.Task.GetBase().Metadata) == 0 {
			continue
		}

		if vErr, err := validateSearchAttributes(task, validate, trans); err != nil {
			return nil, err
		} else if vErr != nil {
			vErrs = append(vErrs, vErr...)
		}
	}

	return vErrs, nil
}
