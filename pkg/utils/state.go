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
	"encoding/json"

	swUtils "github.com/serverlessworkflow/sdk-go/v3/impl/utils"
)

type State struct {
	env   map[string]string
	input any
	data  map[string]any
}

func (s *State) init() {
	if s.env == nil {
		s.env = map[string]string{}
	}
	if s.data == nil {
		s.data = map[string]any{}
	}
}

func (s *State) Add(key string, value any) *State {
	s.init()
	s.data[key] = value

	return s
}

func (s *State) AddEnv(env map[string]string) *State {
	s.init()
	s.env = env

	return s
}

func (s *State) AddInput(input any) *State {
	s.init()
	s.input = input

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
	d := swUtils.DeepClone(s.GetData())

	return NewState(d)
}

func (s *State) Delete(key string, value any) *State {
	s.init()
	delete(s.data, key)

	return s
}

func (s *State) GetData() map[string]any {
	s.init()
	return s.data
}

func (s *State) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.GetData())
}

func NewState(data ...map[string]any) *State {
	s := map[string]any{}

	if len(data) == 1 {
		s = data[0]
	}

	return &State{
		data: s,
	}
}
