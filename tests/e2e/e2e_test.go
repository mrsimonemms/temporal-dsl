//go:build e2e

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

package e2e_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path"
	"testing"

	"github.com/mrsimonemms/golang-helpers/temporal"
	"github.com/mrsimonemms/temporal-dsl/pkg/dsl"
	zlog "github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"go.temporal.io/sdk/client"
)

const (
	INPUT_FILE  = "input.json"
	OUTPUT_FILE = "output.json"
)

type testCase struct {
	Name           string
	WorkflowPath   string
	Input          map[string]any
	ExpectedOutput map[string]any
}

type harness struct {
	Binary string
	Cases  []testCase
}

var h *harness

func getFreePort() (port int, err error) {
	var a *net.TCPAddr
	if a, err = net.ResolveTCPAddr("tcp", "localhost:0"); err == nil {
		var l *net.TCPListener
		if l, err = net.ListenTCP("tcp", a); err == nil {
			defer func() {
				err = l.Close()
			}()
			return l.Addr().(*net.TCPAddr).Port, nil
		}
	}
	return port, err
}

func setup() (*harness, error) {
	// Build the binary
	binaryFile, err := os.MkdirTemp("", "tdsl")
	if err != nil {
		return nil, fmt.Errorf("error creating temp file: %w", err)
	}

	cmd := exec.CommandContext(context.Background(), "go", "build", "-o", binaryFile, "../../")
	if _, err := cmd.Output(); err != nil {
		return nil, fmt.Errorf("error building binary: %w", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("error getting working directory: %w", err)
	}
	testDir := path.Join(wd, "testdata", "tests")

	// Load the test cases
	cases := []testCase{}
	entries, err := os.ReadDir(testDir)
	if err != nil {
		return nil, fmt.Errorf("error reading directories: %w", err)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no test cases found")
	}

	for _, e := range entries {
		i, err := e.Info()
		if err != nil {
			return nil, fmt.Errorf("error getting file info: %w", err)
		}
		if !i.IsDir() {
			return nil, fmt.Errorf("not a directory: %s", i.Name())
		}

		testCaseDir := path.Join(testDir, i.Name())

		a, err := os.ReadFile(path.Join(testCaseDir, INPUT_FILE))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				a = []byte("{}")
			} else {
				return nil, fmt.Errorf("error reading input file: %w", err)
			}
		}

		var input map[string]any
		if err := json.Unmarshal(a, &input); err != nil {
			return nil, fmt.Errorf("error unmarshalling input: %w", err)
		}

		b, err := os.ReadFile(path.Join(testCaseDir, OUTPUT_FILE))
		if err != nil {
			return nil, fmt.Errorf("error reading output file: %w", err)
		}

		var expectedOutput map[string]any
		if err := json.Unmarshal(b, &expectedOutput); err != nil {
			return nil, fmt.Errorf("error unmarshalling output: %w", err)
		}

		cases = append(cases, testCase{
			Name:           i.Name(),
			WorkflowPath:   path.Join(testCaseDir, "workflow.yaml"),
			Input:          input,
			ExpectedOutput: expectedOutput,
		})
	}

	return &harness{
		Binary: path.Join(binaryFile, "temporal-dsl"),
		Cases:  cases,
	}, nil
}

func TestMain(m *testing.M) {
	testHarness, err := setup()
	if err != nil {
		log.Printf("e2e setup failed: %v", err)
		// Non-zero exit so the test run fails clearly.
		os.Exit(1)
	}
	h = testHarness

	code := m.Run()
	os.Exit(code)
}

func TestE2E(t *testing.T) {
	if h == nil {
		t.Fatal("harness is nil - setup not run")
	}

	cancellableCtx := t.Context()
	defer cancellableCtx.Done()

	for _, test := range h.Cases {
		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			workflowDefinition, err := dsl.LoadFromFile(test.WorkflowPath)
			assert.NoError(t, err)

			healthPort, err := getFreePort()
			assert.NoError(t, err, "health port")

			metricsPort, err := getFreePort()
			assert.NoError(t, err, "metrics port")

			args := []string{
				"--file", test.WorkflowPath,
				"--health-listen-address", fmt.Sprintf("localhost:%d", healthPort),
				"--metrics-listen-address", fmt.Sprintf("localhost:%d", metricsPort),
			}

			// Start the Temporal DSL binary with the loaded workflow
			go (func() {
				//nolint
				cmd := exec.CommandContext(cancellableCtx, h.Binary, args...)
				assert.NoError(t, cmd.Run())
			})()

			c, err := temporal.NewConnectionWithEnvvars(
				temporal.WithZerolog(&zlog.Logger),
			)
			assert.NoError(t, err)
			defer c.Close()

			workflowOptions := client.StartWorkflowOptions{
				TaskQueue: workflowDefinition.Document.Namespace,
			}

			wCtx := context.Background()

			we, err := c.ExecuteWorkflow(wCtx, workflowOptions, workflowDefinition.Document.Name, test.Input)
			assert.NoError(t, err)

			var result map[string]any
			assert.NoError(t, we.Get(wCtx, &result))
			assert.Equal(t, test.ExpectedOutput, result)
		})
	}
}
