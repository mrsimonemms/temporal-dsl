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

func Validate(wf *model.Workflow) ([]ValidationErrors, error) {
	enTrans := en.New()
	uni := ut.New(enTrans)
	trans, _ := uni.GetTranslator(enTrans.Locale())

	validate := model.GetValidator()

	if err := en_translations.RegisterDefaultTranslations(validate, trans); err != nil {
		return nil, fmt.Errorf("error registering validator translations: %w", err)
	}

	// Store validation errors
	var vErrs []ValidationErrors

	// Check the workflow
	if err := validate.Struct(wf); err != nil {
		if validationError, ok := err.(validator.ValidationErrors); !ok {
			return nil, ErrUnknownValidationError
		} else {
			for _, v := range validationError {
				vErrs = append(vErrs, ValidationErrors{
					Key:     v.Tag(),
					Message: v.Translate(trans),
				})
			}
		}
	}

	return vErrs, nil
}
