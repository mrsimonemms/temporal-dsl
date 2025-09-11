package run

import (
	"fmt"
	"os/exec"

	"github.com/your-org/sw/internal/runtime"
)

type RunTask struct{}

func New() *RunTask { return &RunTask{} }

func (r *RunTask) Kind() string { return "run.process" }

func (r *RunTask) Execute(ctx *runtime.ExecutionCtx, task map[string]any) (any, error) {
	conf, ok := task["run"].(map[string]any)["process"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("malformed run.process task")
	}
	cmdStr := conf["command"].(string)
	cmd := exec.Command("sh", "-c", cmdStr)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	return string(out), nil
}
