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
	"strconv"
	"strings"
	"time"

	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"github.com/go-viper/mapstructure/v2"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type SearchAttribute struct {
	Type  string `json:"type" validate:"required,oneofci='Datetime KeywordList Text Keyword Int Double Bool"`
	Value any    `json:"value"` // If nil then the value is unset
}

func (v *SearchAttribute) newBooleanUpdate(key string) (temporal.SearchAttributeUpdate, error) {
	s := temporal.NewSearchAttributeKeyBool(key)
	if v.Value == nil {
		return s.ValueUnset(), nil
	}
	switch e := v.Value.(type) {
	case bool:
		return s.ValueSet(e), nil
	case string:
		i, err := strconv.ParseBool(e)
		if err != nil {
			return nil, fmt.Errorf("error converting string to bool")
		}
		return s.ValueSet(i), nil
	default:
		return nil, ErrInvalidType
	}
}

func (v *SearchAttribute) newDateTimeUpdate(key string) (temporal.SearchAttributeUpdate, error) {
	s := temporal.NewSearchAttributeKeyTime(key)
	if v.Value == nil {
		return s.ValueUnset(), nil
	}

	switch e := v.Value.(type) {
	case time.Time:
		return s.ValueSet(e), nil
	case string:
		t, err := time.Parse(time.RFC3339, e)
		if err != nil {
			return nil, fmt.Errorf("error parsing datetime string: %w", err)
		}
		return s.ValueSet(t), nil
	default:
		return nil, ErrInvalidType
	}
}

func (v *SearchAttribute) newFloatUpdate(key string) (temporal.SearchAttributeUpdate, error) {
	s := temporal.NewSearchAttributeKeyFloat64(key)
	if v.Value == nil {
		return s.ValueUnset(), nil
	}

	var val float64
	switch e := v.Value.(type) {
	case int:
		val = float64(e)
	case int32:
		val = float64(e)
	case int64:
		val = float64(e)
	case float32:
		val = float64(e)
	case float64:
		val = float64(e)
	case string:
		var err error
		val, err = strconv.ParseFloat(e, 64)
		if err != nil {
			return nil, fmt.Errorf("error converting string to float64: %w", err)
		}
	default:
		return nil, ErrInvalidType
	}

	return s.ValueSet(val), nil
}

func (v *SearchAttribute) newIntegerUpdate(key string) (temporal.SearchAttributeUpdate, error) {
	s := temporal.NewSearchAttributeKeyInt64(key)
	if v.Value == nil {
		return s.ValueUnset(), nil
	}

	var val int64
	switch e := v.Value.(type) {
	case int:
		val = int64(e)
	case int32:
		val = int64(e)
	case int64:
		val = e
	case float32:
		val = int64(e)
	case float64:
		val = int64(e)
	case string:
		var err error
		val, err = strconv.ParseInt(e, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error converting string to int64: %w", err)
		}
	default:
		return nil, ErrInvalidType
	}

	return s.ValueSet(val), nil
}

func (v *SearchAttribute) newKeywordListUpdate(key string) temporal.SearchAttributeUpdate {
	s := temporal.NewSearchAttributeKeyKeywordList(key)
	if v.Value == nil {
		return s.ValueUnset()
	}
	return s.ValueSet(v.Value.([]string))
}

func (v *SearchAttribute) newKeywordUpdate(key string) temporal.SearchAttributeUpdate {
	s := temporal.NewSearchAttributeKeyKeyword(key)
	if v.Value == nil {
		return s.ValueUnset()
	}
	return s.ValueSet(v.Value.(string))
}

func (v *SearchAttribute) newTextUpdate(key string) temporal.SearchAttributeUpdate {
	s := temporal.NewSearchAttributeKeyString(key)
	if v.Value == nil {
		return s.ValueUnset()
	}
	return s.ValueSet(v.Value.(string))
}

// Sets by type. See the Temporal documentation for what these all mean
// @link https://docs.temporal.io/search-attribute#custom-search-attribute-limits
func (v *SearchAttribute) setAttribute(key string) (temporal.SearchAttributeUpdate, error) {
	switch strings.ToLower(v.Type) {
	case SearchAttributeBooleanType:
		// Boolean
		return v.newBooleanUpdate(key)

	case SearchAttributeDateTimeType:
		// DateTime
		return v.newDateTimeUpdate(key)

	case SearchAttributeDoubleType:
		// Floating point number
		return v.newFloatUpdate(key)

	case SearchAttributeIntType:
		// Integer
		return v.newIntegerUpdate(key)

	case SearchAttributeKeywordType:
		// Keyword
		return v.newKeywordUpdate(key), nil

	case SearchAttributeKeywordListType:
		// Keyword List
		return v.newKeywordListUpdate(key), nil

	case SearchAttributeTextType:
		// Text
		return v.newTextUpdate(key), nil

	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownSearchAttributeType, v.Type)
	}
}

func ParseSearchAttributes(ctx workflow.Context, task *model.TaskBase, vars *Variables) error {
	logger := workflow.GetLogger(ctx)

	if len(task.Metadata) == 0 {
		// No metadata set - continue
		return nil
	}

	search, ok := task.Metadata[MetadataSearchAttribute]
	if !ok {
		// No search attributes
		return nil
	}

	var searchAttributes map[string]*SearchAttribute
	if err := mapstructure.Decode(search, &searchAttributes); err != nil {
		logger.Error("Error converting attributes to golang struct", "error", err)
		return fmt.Errorf("error converting attributes to golang struct: %w", err)
	}

	signedAttributes := make([]temporal.SearchAttributeUpdate, 0)

	for k, v := range searchAttributes {
		if attr, err := v.setAttribute(k); err != nil {
			logger.Error("Error setting search attribute", "error", err)
			return fmt.Errorf("error setting search attribute: %w", err)
		} else {
			signedAttributes = append(signedAttributes, attr)
		}
	}

	if len(signedAttributes) == 0 {
		return nil
	}

	logger.Debug("setting search attribute")
	if err := workflow.UpsertTypedSearchAttributes(ctx, signedAttributes...); err != nil {
		logger.Error("Error upserting search attributes", "error", err)
		return fmt.Errorf("error upserting search attributes: %w", err)
	}

	return nil
}

func validateSearchAttributes(task *model.TaskItem, validate *validator.Validate, trans ut.Translator) ([]ValidationErrors, error) {
	metadata := task.Task.GetBase().Metadata

	if len(metadata) == 0 {
		// No metadata set
		return nil, nil
	}

	search, ok := metadata[MetadataSearchAttribute]
	if !ok {
		// No search attributes
		return nil, nil
	}

	var searchAttributes map[string]*SearchAttribute
	if err := mapstructure.Decode(search, &searchAttributes); err != nil {
		return nil, err
	}

	var vErrs []ValidationErrors

	for key, value := range searchAttributes {
		if err := validate.Struct(value); err != nil {
			if validationError, ok := err.(validator.ValidationErrors); !ok {
				return nil, fmt.Errorf("%w: %s", ErrUnknownValidationError, key)
			} else {
				for _, v := range validationError {
					vErrs = append(vErrs, ValidationErrors{
						Key:     fmt.Sprintf("%s.metadata.%s.%s", task.Key, MetadataSearchAttribute, key),
						Message: v.Translate(trans),
					})
				}
			}
		}
	}

	return vErrs, nil
}
