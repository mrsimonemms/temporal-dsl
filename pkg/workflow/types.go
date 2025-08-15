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

package workflow

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/serverlessworkflow/sdk-go/v3/parser"
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

func (w *Workflow) Activities() *activities {
	return &activities{}
}

func (w *Workflow) WorkflowName() string {
	return w.wf.Document.Name
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
		return fmt.Errorf("%w: run", ErrUnsupportedTask)
	}
	if switchTask := task.AsSwitchTask(); switchTask != nil {
		return fmt.Errorf("%w: switch", ErrUnsupportedTask)
	}
	if try := task.AsTryTask(); try != nil {
		return fmt.Errorf("%w: try", ErrUnsupportedTask)
	}
	return nil
}

func (w *Workflow) Validate() error {
	for _, task := range *w.wf.Do {
		if err := validateTaskSupported(task); err != nil {
			return err
		}
	}

	return nil
}

func LoadFromFile(file, envPrefix string) (*Workflow, error) {
	data, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		return nil, fmt.Errorf("error loading file: %w", err)
	}

	wf, err := parser.FromYAMLSource(data)
	if err != nil {
		return nil, fmt.Errorf("error loading yaml: %w", err)
	}

	// Only support dsl v1.0.0 - we may support later versions
	if dsl := wf.Document.DSL; dsl != "1.0.0" {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedDSL, dsl)
	}

	return &Workflow{
		data:      data,
		envPrefix: strings.ToUpper(envPrefix),
		wf:        wf,
	}, nil
}
