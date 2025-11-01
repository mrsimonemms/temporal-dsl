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

package tasks

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mrsimonemms/temporal-dsl/pkg/utils"
	swUtil "github.com/serverlessworkflow/sdk-go/v3/impl/utils"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

func init() {
	activities = append(activities, callHTTPActivity)
}

// @link: https://github.com/serverlessworkflow/specification/blob/main/dsl-reference.md#http-response
type HTTPResponse struct {
	Request    HTTPRequest       `json:"request"`
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers,omitempty"`
	Content    any               `json:"content,omitempty"`
}

// @link: https://github.com/serverlessworkflow/specification/blob/main/dsl-reference.md#http-request
type HTTPRequest struct {
	Method  string            `json:"method"`
	URI     string            `json:"uri"`
	Headers map[string]string `json:"headers,omitempty"`
}

func NewCallHTTPTaskBuilder(temporalWorker worker.Worker, task *model.CallHTTP, taskName string) (*CallHTTPTaskBuilder, error) {
	return &CallHTTPTaskBuilder{
		builder: builder[*model.CallHTTP]{
			name:           taskName,
			task:           task,
			temporalWorker: temporalWorker,
		},
	}, nil
}

type CallHTTPTaskBuilder struct {
	builder[*model.CallHTTP]
}

func (t *CallHTTPTaskBuilder) Build() (TemporalWorkflowFunc, error) {
	return func(ctx workflow.Context, input any, state *utils.State) (any, error) {
		logger := workflow.GetLogger(ctx)
		logger.Debug("Calling HTTP endpoint", "name", t.name)

		var res any
		if err := workflow.ExecuteActivity(ctx, callHTTPActivity, t.task, input, state).Get(ctx, &res); err != nil {
			if temporal.IsCanceledError(err) {
				return nil, nil
			}

			logger.Error("Error calling HTTP task", "name", t.name, "error", err)
			return nil, fmt.Errorf("error calling http task: %w", err)
		}

		// Add the result to the state's data
		logger.Debug("Setting data to the state", "key", t.name)
		state.AddData(map[string]any{
			t.name: res,
		})

		return res, nil
	}, nil
}

func callHTTPAction(ctx context.Context, task *model.CallHTTP, timeout time.Duration, state *utils.State) (
	resp *http.Response,
	method, url string,
	reqHeaders map[string]string,
	err error,
) {
	logger := activity.GetLogger(ctx)

	args, err := parseHTTPArguments(task, state)
	if err != nil {
		return resp,
			method, url,
			reqHeaders,
			err
	}

	method = strings.ToUpper(args.Method)
	url = args.Endpoint.String()
	body := args.Body

	logger.Debug("Making HTTP call", "method", method, "url", url)
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(body))
	if err != nil {
		logger.Error("Error making HTTP request", "method", method, "url", url, "error", err)
		return resp, method, url, reqHeaders, err
	}

	// Add in headers
	reqHeaders = map[string]string{}
	for k, v := range args.Headers {
		req.Header.Add(k, v)
		reqHeaders[k] = v
	}

	// Add in query strings
	q := req.URL.Query()
	for k, v := range args.Query {
		q.Add(k, v.(string))
	}
	req.URL.RawQuery = q.Encode()

	client := &http.Client{
		Timeout: timeout,
	}

	if !args.Redirect {
		client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	resp, err = client.Do(req)
	if err != nil {
		return resp, method, url, reqHeaders, err
	}

	return resp, method, url, reqHeaders, err
}

func callHTTPActivity(ctx context.Context, task *model.CallHTTP, input any, state *utils.State) (any, error) {
	logger := activity.GetLogger(ctx)
	logger.Debug("Running call HTTP activity")

	info := activity.GetInfo(ctx)

	resp, method, url, reqHeaders, err := callHTTPAction(ctx, task, info.StartToCloseTimeout, state)
	if err != nil {
		logger.Error("Error making HTTP call", "method", method, "url", url, "error", err)
		return nil, err
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
		return nil, err
	}

	// Try converting the body as JSON, returning as string if not possible
	var content any
	var bodyJSON map[string]any
	if err := json.Unmarshal(bodyRes, &bodyJSON); err != nil {
		// Log error
		logger.Debug("Error converting body to JSON", "error", err)
		content = string(bodyRes)
	} else {
		content = bodyJSON
	}

	// Treat redirects as an error - if you have "redirect = true", this will be ignored
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		logger.Error("CallHTTP returned 3xx status")
		return nil, temporal.NewNonRetryableApplicationError(
			"CallHTTP returned 3xx status code",
			"CallHTTP error",
			errors.New(resp.Status),
			content,
		)
	}

	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		// Client error - treat as non-retryable error as we need to fix it
		logger.Error("CallHTTP returned 4xx error")
		return nil, temporal.NewNonRetryableApplicationError(
			"CallHTTP returned 4xx status code",
			"CallHTTP error",
			errors.New(resp.Status),
			content,
		)
	}

	respHeader := map[string]string{}
	for k, v := range resp.Header {
		respHeader[k] = strings.Join(v, ", ")
	}

	httpResponse := HTTPResponse{
		Request: HTTPRequest{
			Method:  method,
			URI:     url,
			Headers: reqHeaders,
		},
		StatusCode: resp.StatusCode,
		Headers:    respHeader,
		Content:    content,
	}

	return parseOutput(task.With.Output, httpResponse, bodyRes), err
}

// parseHTTPArguments note that I looked at the github.com/go-viper/mapstructure/v2.Decode
// function, but this wasn't able to decode some of the more complex data types. This is
// more heavyweight than I'd like, but it's fine for now.
func parseHTTPArguments(task *model.CallHTTP, state *utils.State) (*model.HTTPArguments, error) {
	// First, we need to convert it to map[string]any
	b, err := json.Marshal(task.With)
	if err != nil {
		return nil, fmt.Errorf("error marshalling object to bytes: %w", err)
	}

	// Next, convert it to a map so we can traverse
	var data map[string]any
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, fmt.Errorf("error unmarshalling data to map: %w", err)
	}

	// Clone and traverse, interpolating the data
	cloneData := swUtil.DeepClone(data)
	obj, err := utils.TraverseAndEvaluateObj(model.NewObjectOrRuntimeExpr(cloneData), state)
	if err != nil {
		return nil, fmt.Errorf("error traversing http data object: %w", err)
	}

	// Now, put it back to a JSON string
	e, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("error marshalling object to bytes: %w", err)
	}

	// Finally, convert back to HTTPArguments
	var result model.HTTPArguments
	if err := json.Unmarshal(e, &result); err != nil {
		return nil, fmt.Errorf("error unmarshalling data to map: %w", err)
	}

	return &result, nil
}

func parseOutput(outputType string, httpResp HTTPResponse, raw []byte) any {
	var output any
	switch outputType {
	case "raw":
		// Base64 encoded HTTP response content - use the bodyRes
		output = base64.StdEncoding.EncodeToString(raw)
	case "response":
		// HTTP response
		output = httpResp
	default:
		// Content
		output = httpResp.Content
	}

	return output
}
