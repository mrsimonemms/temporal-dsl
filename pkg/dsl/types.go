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
	"maps"

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"go.temporal.io/sdk/workflow"
)

type activities struct{}

type Workflow struct {
	data      []byte
	envPrefix string
	wf        *model.Workflow
}

type OutputType struct {
	Type ResultType `json:"type"`
	Data any        `json:"data"`
}

type HTTPData map[string]any

type Variables struct {
	Data HTTPData `json:"data"`
}

func (a *Variables) AddData(d HTTPData) {
	if a.Data == nil {
		a.Data = make(HTTPData)
	}

	maps.Copy(a.Data, d)
}

func (a *Variables) Clone() *Variables {
	if a.Data == nil {
		a.Data = make(HTTPData)
	}

	return &Variables{
		Data: maps.Clone(a.Data),
	}
}

type CancellableFuture struct {
	Cancel  workflow.CancelFunc
	Context workflow.Context
	Future  workflow.ChildWorkflowFuture
}
