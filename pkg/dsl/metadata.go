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

	"github.com/go-viper/mapstructure/v2"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type SearchAttribute struct {
	Type  string `json:"type" validate:"required,oneofci='Datetime KeywordList Text Keyword Int Double Bool"`
	Value any    `json:"value"` // If nil then the value is unset
}

type searchAttributeInterface interface {
	ValueUnset() temporal.SearchAttributeUpdate
}

func (v *SearchAttribute) setValue(
	key searchAttributeInterface,
	value any, s func() (temporal.SearchAttributeUpdate, error),
) (temporal.SearchAttributeUpdate, error) {
	if value == nil {
		return key.ValueUnset(), nil
	} else {
		return s()
	}
}

func (v *SearchAttribute) UpdateAttr(k string) (attr temporal.SearchAttributeUpdate, err error) {
	switch strings.ToLower(v.Type) {
	case "datetime":
		key := temporal.NewSearchAttributeKeyTime(k)
		attr, err = v.setValue(key, v.Value, func() (temporal.SearchAttributeUpdate, error) {
			return key.ValueSet(v.Value.(time.Time)), nil
		})
	case "keywordlist":
		key := temporal.NewSearchAttributeKeyKeywordList(k)
		attr, err = v.setValue(key, v.Value, func() (temporal.SearchAttributeUpdate, error) {
			return key.ValueSet(v.Value.([]string)), nil
		})
	case "keyword":
		key := temporal.NewSearchAttributeKeyKeyword(k)
		attr, err = v.setValue(key, v.Value, func() (temporal.SearchAttributeUpdate, error) {
			return key.ValueSet(v.Value.(string)), nil
		})
	case "text":
		key := temporal.NewSearchAttributeKeyString(k)
		attr, err = v.setValue(key, v.Value, func() (temporal.SearchAttributeUpdate, error) {
			return key.ValueSet(v.Value.(string)), nil
		})
	case "int":
		key := temporal.NewSearchAttributeKeyInt64(k)
		attr, err = v.setValue(key, v.Value, func() (temporal.SearchAttributeUpdate, error) {
			var val int64
			var err error
			switch v.Value.(type) {
			case string:
				val, err = strconv.ParseInt(v.Value.(string), 10, 64)
				if err != nil {
					return nil, err
				}
				return v.setValue(key, val, func() (temporal.SearchAttributeUpdate, error) {
					return key.ValueSet(val), nil
				})
			case float64:
			default:
				return nil, fmt.Errorf("pants")
			}
			return key.ValueSet(val), nil
		})
	case "double":
		key := temporal.NewSearchAttributeKeyFloat64(k)
		attr, err = v.setValue(key, v.Value, func() (temporal.SearchAttributeUpdate, error) {
			return key.ValueSet(v.Value.(float64)), nil
		})
	case "bool":
		key := temporal.NewSearchAttributeKeyBool(k)
		attr, err = v.setValue(key, v.Value, func() (temporal.SearchAttributeUpdate, error) {
			return key.ValueSet(v.Value.(bool)), nil
		})
	default:
		err = fmt.Errorf("%w: %s", ErrUnknownListenTypeTask, v.Type)
	}

	return attr, err
}

func validateMetadata(task *model.TaskItem) error {
	metadata := task.Task.GetBase().Metadata
	if len(metadata) == 0 {
		// No metadata set - continue
		return nil
	}

	return nil
}

func ParseSearchAttributes(ctx workflow.Context, task *model.TaskBase, vars *Variables) error {
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
		return err
	}

	signedAttributes := make([]temporal.SearchAttributeUpdate, 0)

	for k, v := range searchAttributes {
		if attr, err := v.UpdateAttr(k); err != nil {
			return fmt.Errorf("error setting search attribute: %w", err)
		} else {
			signedAttributes = append(signedAttributes, attr)
		}
	}

	if err := workflow.UpsertTypedSearchAttributes(ctx, signedAttributes...); err != nil {
		return fmt.Errorf("error upserting search attributes: %w", err)
	}

	return nil
}
