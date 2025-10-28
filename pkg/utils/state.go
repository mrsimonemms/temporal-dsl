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

package utils

import (
	swUtils "github.com/serverlessworkflow/sdk-go/v3/impl/utils"
)

type State struct {
	Data   map[string]any `json:"data"`            // Data stored along the way
	Env    map[string]any `json:"env"`             // Available environment variables
	Input  any            `json:"input,omitempty"` // The input given by the caller
	Output map[string]any `json:"output"`          // What will be output to the caller
}

func (s *State) init() *State {
	if s.Env == nil {
		s.Env = map[string]any{}
	}
	if s.Data == nil {
		s.Data = map[string]any{}
	}
	if s.Output == nil {
		s.Output = map[string]any{}
	}

	return s
}

func (s *State) Clone() *State {
	s1 := NewState()

	s1.Data = swUtils.DeepClone(s.Data)
	s1.Env = swUtils.DeepClone(s1.Env)
	s1.Input = swUtils.DeepCloneValue(s.Input)
	s1.Output = swUtils.DeepClone(s1.Output)

	return s1
}

// Returns the state as a map. This can be used for
func (s *State) GetAsMap() map[string]any {
	s1 := s.Clone()

	return map[string]any{
		"data":   s1.Data,
		"env":    s1.Env,
		"input":  s1.Input,
		"output": s1.Output,
	}
}

func NewState() *State {
	s := &State{}
	return s.init()
}
