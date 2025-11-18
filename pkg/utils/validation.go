/*
 * Copyright 2025 Zigflow authors <https://github.com/mrsimonemms/zigflow/graphs/contributors>
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

package utils

import (
	"fmt"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	"github.com/serverlessworkflow/sdk-go/v3/model"
)

var ErrUnknownValidationError = fmt.Errorf("unknown validation error")

type ValidationErrors struct {
	Key     string
	Message string
}

type Validator struct {
	validate *validator.Validate
	trans    ut.Translator
}

func (v *Validator) ValidateStruct(data any) ([]ValidationErrors, error) {
	// Store validation errors
	var vErrs []ValidationErrors

	// Check the data
	if err := v.validate.Struct(data); err != nil {
		if validationError, ok := err.(validator.ValidationErrors); !ok {
			return nil, fmt.Errorf("%s: %w", ErrUnknownValidationError, err)
		} else {
			for _, e := range validationError {
				vErrs = append(vErrs, ValidationErrors{
					Key:     e.Tag(),
					Message: e.Translate(v.trans),
				})
			}
		}
	}

	return vErrs, nil
}

func NewValidator() (*Validator, error) {
	enTrans := en.New()
	uni := ut.New(enTrans)
	trans, _ := uni.GetTranslator(enTrans.Locale())

	validate := model.GetValidator()

	if err := en_translations.RegisterDefaultTranslations(validate, trans); err != nil {
		return nil, fmt.Errorf("error registering validator translations: %w", err)
	}

	return &Validator{
		validate: validate,
	}, nil
}
