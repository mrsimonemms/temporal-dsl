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
	"fmt"
	"maps"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/serverlessworkflow/sdk-go/v3/impl/utils"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type TemporalWorkflowTask struct {
	Key      string
	TaskBase *model.TaskBase
	Task     TemporalWorkflowFunc
}

type TemporalWorkflowFunc func(ctx workflow.Context, data *Variables, output map[string]OutputType) error

type TemporalWorkflow struct {
	EnvPrefix string
	Name      string
	Timeout   time.Duration
	Tasks     []TemporalWorkflowTask
	workflow  *model.Workflow
}

func (t *TemporalWorkflow) validateInput(ctx workflow.Context, input HTTPData) error {
	logger := workflow.GetLogger(ctx)

	if t.workflow.Input != nil {
		logger.Debug("Validating input against schema")
		if err := utils.ValidateSchema(input, t.workflow.Input.Schema, t.Name); err != nil {
			logger.Error("Input failed data validation", "error", err)

			return temporal.NewNonRetryableApplicationError(
				"Workflow input did not meet JSON schema specification",
				"Validation",
				err,
				// There is additional detail useful in here
				err.(*model.Error),
			)
		}
	}

	return nil
}

func (t *TemporalWorkflow) Workflow(ctx workflow.Context, input HTTPData) (map[string]OutputType, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Running workflow")

	if err := t.validateInput(ctx, input); err != nil {
		return nil, err
	}

	logger.Debug("Setting workflow options", "StartToCloseTimeout", t.Timeout)
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: t.Timeout,
	})

	vars := &Variables{
		Data: GetWorkflowInfo(ctx),
	}
	maps.Copy(vars.Data, input)
	output := map[string]OutputType{}

	// Load in any envvars with the prefix
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if strings.HasPrefix(pair[0], t.EnvPrefix) {
			vars.Data[pair[0]] = pair[1]
		}
	}

	for _, task := range t.Tasks {
		logger.Debug("Adding summary to activity context", "name", task.Key)
		ao := workflow.GetActivityOptions(ctx)
		ao.Summary = task.Key
		ctx = workflow.WithActivityOptions(ctx, ao)

		// Set task key to the variable
		vars.AddData(HTTPData{
			"_task_key": task.Key,
		})

		logger.Debug("Check if task can be run", "name", task.Key)
		// Check for and run any if statement
		if toRun, err := CheckIfStatement(task.TaskBase.If, vars); err != nil {
			logger.Error("Error checking if statement", "error", err)
			return nil, err
		} else if !toRun {
			logger.Debug("Skipping task as if statement resolved as false", "name", task.Key)
			continue
		}

		// Parse any custom search attributes
		if err := ParseSearchAttributes(ctx, task.TaskBase, vars); err != nil {
			logger.Error("Error parsing search attributes", "error", err)
			return nil, err
		}

		logger.Info("Running task", "name", task.Key)
		if err := task.Task(ctx, vars, output); err != nil {
			return nil, err
		}
	}

	return output, nil
}

// buildWorkflowTask convert the individual tasks to Temporal
func (w *Workflow) buildWorkflowTask(item *model.TaskItem) (
	task TemporalWorkflowFunc,
	taskType string,
	additionalWorkflows []*TemporalWorkflow,
	err error,
) {
	if do := item.AsDoTask(); do != nil {
		additionalWorkflows, err = doTaskImpl(do, item, w)
		taskType = "DoTask"
	}

	if fork := item.AsForkTask(); fork != nil {
		task, err = forkTaskImpl(fork, item, w)
		taskType = "ForkTask"
	}

	if http := item.AsCallHTTPTask(); http != nil {
		task = httpTaskImpl(http, item.Key)
		taskType = "CallHTTP"
	}

	if listen := item.AsListenTask(); listen != nil {
		task, err = listenTaskImpl(listen, item.Key)
		taskType = "ListenTask"
	}

	if raise := item.AsRaiseTask(); raise != nil {
		task = raiseTaskImpl(raise, item.Key)
		taskType = "RaiseTask"
	}

	if run := item.AsRunTask(); run != nil {
		task, err = runTaskImpl(run, item.Key)
		taskType = "RunTask"
	}

	if set := item.AsSetTask(); set != nil {
		task = setTaskImpl(set)
		taskType = "SetTask"
	}

	if switchTask := item.AsSwitchTask(); switchTask != nil {
		task, err = setSwitchImpl(switchTask, item.Key)
		taskType = "SwitchTask"
	}

	if wait := item.AsWaitTask(); wait != nil {
		task = waitTaskImpl(wait)
		taskType = "WaitTask"
	}

	return task,
		taskType,
		additionalWorkflows,
		err
}

func (w *Workflow) workflowBuilder(tasks *model.TaskList, name string) ([]*TemporalWorkflow, error) {
	wfs := make([]*TemporalWorkflow, 0)

	timeout := defaultWorkflowTimeout
	if w.wf.Timeout != nil && w.wf.Timeout.Timeout != nil && w.wf.Timeout.Timeout.After != nil {
		timeout = ToDuration(w.wf.Timeout.Timeout.After)
	}

	wf := &TemporalWorkflow{
		EnvPrefix: w.envPrefix,
		Name:      name,
		Tasks:     make([]TemporalWorkflowTask, 0),
		Timeout:   timeout,
		workflow:  w.wf,
	}

	var hasNoDo bool

	// Iterate over the task list to build out our workflow(s)
	for _, item := range *tasks {
		if do := item.AsDoTask(); do == nil {
			hasNoDo = true
		}

		task, taskType, additionalWorkflows, err := w.buildWorkflowTask(item)
		if err != nil {
			return nil, err
		}

		// Register additional workflows
		wfs = append(wfs, additionalWorkflows...)

		l := log.With().Str("key", item.Key).Logger()
		if taskType != "" {
			l.Debug().Str("type", taskType).Msg("Task detected")
		} else {
			l.Warn().Msg("Task detected, but no taskType set")
		}

		if task != nil {
			wf.Tasks = append(wf.Tasks, TemporalWorkflowTask{
				Key:      item.Key,
				TaskBase: item.GetBase(),
				Task:     task,
			})
		}
	}

	// Add to the list of workflows
	if hasNoDo {
		wfs = append(wfs, wf)
	} else {
		log.Debug().Str("workflow", name).Msg("Workflow exclusively made of Do tasks - not registering as workflow")
	}

	return wfs, nil
}

// This is the main workflow definition.
func (w *Workflow) BuildWorkflows() ([]*TemporalWorkflow, error) {
	wfs := make([]*TemporalWorkflow, 0)

	d, err := w.workflowBuilder(w.wf.Do, w.WorkflowName())
	if err != nil {
		return nil, fmt.Errorf("error building workflows: %w", err)
	}

	wfs = append(wfs, d...)
	return wfs, nil
}

func NewWorkflow(wf *model.Workflow, data []byte, envPrefix string) *Workflow {
	return &Workflow{
		data:      data,
		envPrefix: envPrefix,
		wf:        wf,
	}
}
