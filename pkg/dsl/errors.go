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

import "fmt"

var (
	ErrDuplicateKey               = fmt.Errorf("duplicate key found")
	ErrInvalidType                = fmt.Errorf("invalid type given")
	ErrScheduleNoWorkflowName     = fmt.Errorf("workflow name not set for schedule")
	ErrNotString                  = fmt.Errorf("input must be a string")
	ErrUnknownValidationError     = fmt.Errorf("unknown validation error")
	ErrUnsetListenIDTask          = fmt.Errorf("listen task id is not set")
	ErrUnsetListenTypeTask        = fmt.Errorf("listen task type is not set")
	ErrUnknownListenTypeTask      = fmt.Errorf("listen task type is not known")
	ErrUnknownSearchAttributeType = fmt.Errorf("unknown search attribute type")
	ErrUnsupportedTask            = fmt.Errorf("task not supported")
	ErrUnsupportedDSL             = fmt.Errorf("unsupported dsl")
)
