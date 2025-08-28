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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"strings"
	"time"

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type CallHTTPResult struct {
	Body       string         `json:"body,omitempty"`
	BodyJSON   map[string]any `json:"bodyJSON,omitempty"`
	Method     string         `json:"method"`
	Status     string         `json:"status"`
	StatusCode int            `json:"statusCode"`
	URL        string         `json:"url"`
}

func parseCallBody(input json.RawMessage, data *Variables) ([]byte, error) {
	// The input might be empty, a single or double-encoded piece of JSON.
	if strings.TrimSpace(string(input)) != "" {
		// It's not empty
		if err := json.Unmarshal(input, &HTTPData{}); err != nil {
			// It's not single-encoded
			var i string
			if err := json.Unmarshal(input, &i); err != nil {
				// It's not double-encoded
				return nil, fmt.Errorf("cannot parse input body: %w", err)
			}
			input = []byte(i)
		}
	}

	d, err := input.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling body: %w", err)
	}
	body, err := ParseVariables(string(d), data)
	if err != nil {
		return nil, fmt.Errorf("error interpolating body: %w", err)
	}

	return []byte(body), nil
}

func (a *activities) CallHTTP(ctx context.Context, callHttp *model.CallHTTP, vars *Variables) (*CallHTTPResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Debug("Running call HTTP activity")

	vars = vars.Clone()
	vars.AddData(GetActivityVars(ctx))

	body, err := parseCallBody(callHttp.With.Body, vars)
	if err != nil {
		return nil, err
	}

	method := strings.ToUpper(MustParseVariables(callHttp.With.Method, vars))
	url := MustParseVariables(callHttp.With.Endpoint.String(), vars)

	logger.Debug("Making HTTP call", "method", method, "url", url)
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(body))
	if err != nil {
		logger.Error("Error making HTTP request", "method", method, "url", url, "error", err)
		return nil, fmt.Errorf("error making http request: %w", err)
	}

	for k, v := range callHttp.With.Headers {
		req.Header.Add(k, MustParseVariables(v, vars))
	}

	q := req.URL.Query()
	for k, v := range callHttp.With.Query {
		q.Add(k, MustParseVariables(v.(string), vars))
	}
	req.URL.RawQuery = q.Encode()

	// @todo(sje): configure the timeout
	client := http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		logger.Error("Error making HTTP call", "method", method, "url", url, "error", err)
		return nil, fmt.Errorf("error making http call: %w", err)
	}
	defer func() {
		err = resp.Body.Close()
		if err != nil {
			logger.Error("Error closing body reader", "error", err)
		}
	}()

	bodyRes, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Error reading HTTP body", "method", method, "url", url, "error", err)
		return nil, fmt.Errorf("error reading http body: %w", err)
	}

	// Try converting the body as JSON, returning as string if not possible
	var bodyJSON map[string]any
	var bodyStr string
	if err := json.Unmarshal(bodyRes, &bodyJSON); err != nil {
		// Log error
		logger.Debug("Error converting body to JSON", "error", err)
		bodyStr = string(bodyRes)
	}

	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		// Error on our side - treat as non-retryable error as we need to fix it
		logger.Error("CallHTTP returned 4xx error")

		return nil, temporal.NewNonRetryableApplicationError(
			"CallHTTP returned 4xx error",
			string(CallHTTPErr),
			errors.New(resp.Status),
			HTTPData{
				"statusCode": resp.StatusCode,
				"body":       bodyStr,
				"json":       bodyJSON,
			},
		)
	}

	if resp.StatusCode >= 500 && resp.StatusCode < 600 {
		// Error on their side - treat as retryable error as we can't fix it
		logger.Error("CallHTTP returned 5xx error")

		return nil, temporal.NewApplicationError("CallHTTP returned 5xx error", string(CallHTTPErr), errors.New(resp.Status), HTTPData{
			"statusCode": resp.StatusCode,
			"body":       bodyStr,
			"json":       bodyJSON,
		})
	}

	return &CallHTTPResult{
		Body:       bodyStr,
		BodyJSON:   bodyJSON,
		Method:     method,
		Status:     resp.Status,
		StatusCode: resp.StatusCode,
		URL:        url,
	}, err
}

func httpTaskImpl(task *model.CallHTTP, key string) TemporalWorkflowFunc {
	var a *activities

	return func(ctx workflow.Context, data *Variables, output map[string]OutputType) error {
		logger := workflow.GetLogger(ctx)
		logger.Debug("Calling HTTP endpoint")

		var result CallHTTPResult
		if err := workflow.ExecuteActivity(ctx, a.CallHTTP, task, data).Get(ctx, &result); err != nil {
			return fmt.Errorf("error calling http task: %w", err)
		}

		maps.Copy(output, map[string]OutputType{
			key: {
				Type: CallHTTPResultType,
				Data: result,
			},
		})

		return nil
	}
}
