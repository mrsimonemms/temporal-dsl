package http

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/your-org/sw/internal/runtime"
)

type HTTPTask struct{}

func New() *HTTPTask { return &HTTPTask{} }

func (h *HTTPTask) Kind() string { return "call.http" }

func (h *HTTPTask) Execute(ctx *runtime.ExecutionCtx, task map[string]any) (any, error) {
	conf, ok := task["call"].(map[string]any)["http"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("malformed http task")
	}
	method := conf["method"].(string)
	url := conf["url"].(string)

	var body []byte
	if b, ok := conf["body"].(string); ok {
		body = []byte(b)
	}

	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := ioutil.ReadAll(resp.Body)
	return string(respBody), nil
}
