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
	Data  map[string]any `json:"data"`
	Env   map[string]any `json:"env"`
	Input any            `json:"input"`
}

func (s *State) init() {
	if s.Env == nil {
		s.Env = map[string]any{}
	}
	if s.Data == nil {
		s.Data = map[string]any{}
	}
}

func (s *State) Add(key string, value any) *State {
	s.init()
	s.Data[key] = value

	return s
}

func (s *State) AddEnv(env map[string]any) *State {
	s.init()
	s.Env = env

	return s
}

func (s *State) AddInput(input any) *State {
	s.init()
	s.Input = input

	return s
}

func (s *State) BulkAdd(data map[string]any) *State {
	s.init()
	for k, v := range data {
		s.Add(k, v)
	}

	return s
}

func (s *State) Clone() *State {
	state := NewState(swUtils.DeepClone(s.Data))
	state.Input = s.Input
	state.Env = swUtils.DeepClone(s.Env)

	return state
}

func (s *State) Delete(key string, value any) *State {
	s.init()
	delete(s.Data, key)

	return s
}

func (s *State) GetData() map[string]any {
	s.init()
	return s.Data
}

func (s *State) ParseData() map[string]any {
	// Clone and get the raw data
	d := s.Clone().GetData()

	// Add in the input and envvars
	d["__input"] = s.Input
	d["__env"] = s.Env

	return d
}

func NewState(data ...map[string]any) *State {
	s := map[string]any{}

	if len(data) == 1 {
		s = data[0]
	}

	return &State{
		Data: s,
	}
}
