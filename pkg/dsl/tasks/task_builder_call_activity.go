/*
 * Copyright 2025 Temporal DSL authors <https://github.com/mrsimonemms/temporal-dsl/graphs/contributors>
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
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mrsimonemms/temporal-dsl/pkg/utils"
	swUtil "github.com/serverlessworkflow/sdk-go/v3/impl/utils"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

func NewCallActivityTaskBuilder(
	temporalWorker worker.Worker,
	task *model.CallFunction,
	taskName string,
	doc *model.Workflow,
) (*CallActivityTaskBuilder, error) {
	if task.Call != "activity" {
		return nil, fmt.Errorf("unsupported call task '%s' for activity builder", task.Call)
	}

	return &CallActivityTaskBuilder{
		builder: builder[*model.CallFunction]{
			doc:            doc,
			name:           taskName,
			task:           task,
			temporalWorker: temporalWorker,
		},
	}, nil
}

type CallActivityTaskBuilder struct {
	builder[*model.CallFunction]
}

func (t *CallActivityTaskBuilder) Build() (TemporalWorkflowFunc, error) {
	return func(ctx workflow.Context, input any, state *utils.State) (any, error) {
		logger := workflow.GetLogger(ctx)

		args, err := parseActivityCallArguments(t.task, state)
		if err != nil {
			return nil, err
		}

		if args.Name == "" {
			return nil, errors.New("call activity requires a name")
		}

		logger.Info("Executing Temporal activity", "activity", args.Name, "task", t.GetTaskName())

		execCtx := ctx
		if args.Local {
			laOpts, err := applyLocalActivityOverrides(workflow.GetLocalActivityOptions(ctx), args.LocalOptions)
			if err != nil {
				return nil, err
			}
			execCtx = workflow.WithLocalActivityOptions(ctx, laOpts)
		} else {
			aOpts, err := applyActivityOverrides(workflow.GetActivityOptions(ctx), args.Options)
			if err != nil {
				return nil, err
			}
			execCtx = workflow.WithActivityOptions(ctx, aOpts)
		}

		var future workflow.Future
		if args.Local {
			future = workflow.ExecuteLocalActivity(execCtx, args.Name, args.Arguments...)
		} else {
			future = workflow.ExecuteActivity(execCtx, args.Name, args.Arguments...)
		}

		var res any
		if err := future.Get(ctx, &res); err != nil {
			if temporal.IsCanceledError(err) {
				logger.Debug("Activity cancelled", "activity", args.Name)
				return nil, nil
			}
			logger.Error("Error executing activity", "activity", args.Name, "error", err)
			return nil, fmt.Errorf("error executing activity %s: %w", args.Name, err)
		}

		state.AddData(map[string]any{
			t.GetTaskName(): res,
		})

		return res, nil
	}, nil
}

type ActivityCallArguments struct {
	Name         string                    `json:"name"`
	Local        bool                      `json:"local,omitempty"`
	Arguments    []any                     `json:"arguments,omitempty"`
	Options      *ActivityCallOptions      `json:"options,omitempty"`
	LocalOptions *LocalActivityCallOptions `json:"localOptions,omitempty"`
}

type ActivityCallOptions struct {
	TaskQueue              string               `json:"taskQueue,omitempty"`
	ScheduleToCloseTimeout *model.Duration      `json:"scheduleToCloseTimeout,omitempty"`
	ScheduleToStartTimeout *model.Duration      `json:"scheduleToStartTimeout,omitempty"`
	StartToCloseTimeout    *model.Duration      `json:"startToCloseTimeout,omitempty"`
	HeartbeatTimeout       *model.Duration      `json:"heartbeatTimeout,omitempty"`
	WaitForCancellation    *bool                `json:"waitForCancellation,omitempty"`
	ActivityID             string               `json:"activityId,omitempty"`
	RetryPolicy            *ActivityRetryPolicy `json:"retryPolicy,omitempty"`
	DisableEagerExecution  *bool                `json:"disableEagerExecution,omitempty"`
	VersioningIntent       string               `json:"versioningIntent,omitempty"`
	Summary                string               `json:"summary,omitempty"`
	Priority               *ActivityPriority    `json:"priority,omitempty"`
}

type LocalActivityCallOptions struct {
	ScheduleToCloseTimeout *model.Duration      `json:"scheduleToCloseTimeout,omitempty"`
	StartToCloseTimeout    *model.Duration      `json:"startToCloseTimeout,omitempty"`
	RetryPolicy            *ActivityRetryPolicy `json:"retryPolicy,omitempty"`
	Summary                string               `json:"summary,omitempty"`
}

type ActivityRetryPolicy struct {
	InitialInterval        *model.Duration `json:"initialInterval,omitempty"`
	BackoffCoefficient     *float64        `json:"backoffCoefficient,omitempty"`
	MaximumInterval        *model.Duration `json:"maximumInterval,omitempty"`
	MaximumAttempts        *int32          `json:"maximumAttempts,omitempty"`
	NonRetryableErrorTypes []string        `json:"nonRetryableErrorTypes,omitempty"`
}

type ActivityPriority struct {
	PriorityKey    *int     `json:"priorityKey,omitempty"`
	FairnessKey    string   `json:"fairnessKey,omitempty"`
	FairnessWeight *float32 `json:"fairnessWeight,omitempty"`
}

func parseActivityCallArguments(task *model.CallFunction, state *utils.State) (*ActivityCallArguments, error) {
	var clone map[string]any
	if task.With != nil {
		clone = swUtil.DeepClone(task.With)
	} else {
		clone = map[string]any{}
	}
	interpolated, err := utils.TraverseAndEvaluateObj(model.NewObjectOrRuntimeExpr(clone), state)
	if err != nil {
		return nil, fmt.Errorf("error interpolating activity call arguments: %w", err)
	}

	payload, err := json.Marshal(interpolated)
	if err != nil {
		return nil, fmt.Errorf("error marshalling activity call arguments: %w", err)
	}

	var args ActivityCallArguments
	if err := json.Unmarshal(payload, &args); err != nil {
		return nil, fmt.Errorf("error unmarshalling activity call arguments: %w", err)
	}

	if args.Arguments == nil {
		args.Arguments = []any{}
	}

	return &args, nil
}

func applyActivityOverrides(base workflow.ActivityOptions, cfg *ActivityCallOptions) (workflow.ActivityOptions, error) {
	if cfg == nil {
		return base, nil
	}

	if cfg.TaskQueue != "" {
		base.TaskQueue = cfg.TaskQueue
	}
	if err := setActivityDurations(&base, cfg); err != nil {
		return base, err
	}
	if cfg.WaitForCancellation != nil {
		base.WaitForCancellation = *cfg.WaitForCancellation
	}
	if cfg.ActivityID != "" {
		base.ActivityID = cfg.ActivityID
	}
	if cfg.RetryPolicy != nil {
		rp, err := convertRetryPolicy(cfg.RetryPolicy)
		if err != nil {
			return base, err
		}
		base.RetryPolicy = rp
	}
	if cfg.DisableEagerExecution != nil {
		base.DisableEagerExecution = *cfg.DisableEagerExecution
	}
	if cfg.VersioningIntent != "" {
		vi, err := parseVersioningIntent(cfg.VersioningIntent)
		if err != nil {
			return base, err
		}
		base.VersioningIntent = vi
	}
	if cfg.Summary != "" {
		base.Summary = cfg.Summary
	}
	if cfg.Priority != nil {
		base.Priority = convertPriority(cfg.Priority)
	}

	return base, nil
}

func setActivityDurations(opts *workflow.ActivityOptions, cfg *ActivityCallOptions) error {
	var err error
	if cfg.ScheduleToCloseTimeout != nil {
		if opts.ScheduleToCloseTimeout, err = durationToTime(cfg.ScheduleToCloseTimeout); err != nil {
			return fmt.Errorf("invalid scheduleToCloseTimeout: %w", err)
		}
	}
	if cfg.ScheduleToStartTimeout != nil {
		if opts.ScheduleToStartTimeout, err = durationToTime(cfg.ScheduleToStartTimeout); err != nil {
			return fmt.Errorf("invalid scheduleToStartTimeout: %w", err)
		}
	}
	if cfg.StartToCloseTimeout != nil {
		if opts.StartToCloseTimeout, err = durationToTime(cfg.StartToCloseTimeout); err != nil {
			return fmt.Errorf("invalid startToCloseTimeout: %w", err)
		}
	}
	if cfg.HeartbeatTimeout != nil {
		if opts.HeartbeatTimeout, err = durationToTime(cfg.HeartbeatTimeout); err != nil {
			return fmt.Errorf("invalid heartbeatTimeout: %w", err)
		}
	}

	return nil
}

func applyLocalActivityOverrides(base workflow.LocalActivityOptions, cfg *LocalActivityCallOptions) (workflow.LocalActivityOptions, error) {
	if cfg == nil {
		return base, nil
	}

	var err error
	if cfg.ScheduleToCloseTimeout != nil {
		if base.ScheduleToCloseTimeout, err = durationToTime(cfg.ScheduleToCloseTimeout); err != nil {
			return base, fmt.Errorf("invalid local scheduleToCloseTimeout: %w", err)
		}
	}
	if cfg.StartToCloseTimeout != nil {
		if base.StartToCloseTimeout, err = durationToTime(cfg.StartToCloseTimeout); err != nil {
			return base, fmt.Errorf("invalid local startToCloseTimeout: %w", err)
		}
	}
	if cfg.RetryPolicy != nil {
		rp, err := convertRetryPolicy(cfg.RetryPolicy)
		if err != nil {
			return base, err
		}
		base.RetryPolicy = rp
	}
	if cfg.Summary != "" {
		base.Summary = cfg.Summary
	}

	return base, nil
}

func convertRetryPolicy(cfg *ActivityRetryPolicy) (*temporal.RetryPolicy, error) {
	if cfg == nil {
		return nil, nil
	}

	retry := &temporal.RetryPolicy{}
	var hasValue bool
	if cfg.InitialInterval != nil {
		d, err := durationToTime(cfg.InitialInterval)
		if err != nil {
			return nil, fmt.Errorf("invalid retry initial interval: %w", err)
		}
		retry.InitialInterval = d
		hasValue = true
	}
	if cfg.BackoffCoefficient != nil {
		retry.BackoffCoefficient = *cfg.BackoffCoefficient
		hasValue = true
	}
	if cfg.MaximumInterval != nil {
		d, err := durationToTime(cfg.MaximumInterval)
		if err != nil {
			return nil, fmt.Errorf("invalid retry maximum interval: %w", err)
		}
		retry.MaximumInterval = d
		hasValue = true
	}
	if cfg.MaximumAttempts != nil {
		retry.MaximumAttempts = *cfg.MaximumAttempts
		hasValue = true
	}
	if len(cfg.NonRetryableErrorTypes) > 0 {
		retry.NonRetryableErrorTypes = cfg.NonRetryableErrorTypes
		hasValue = true
	}

	if !hasValue {
		return nil, nil
	}

	return retry, nil
}

func convertPriority(cfg *ActivityPriority) temporal.Priority {
	priority := temporal.Priority{}
	if cfg.PriorityKey != nil {
		priority.PriorityKey = *cfg.PriorityKey
	}
	if cfg.FairnessKey != "" {
		priority.FairnessKey = cfg.FairnessKey
	}
	if cfg.FairnessWeight != nil {
		priority.FairnessWeight = *cfg.FairnessWeight
	}
	return priority
}

var versioningIntentMap = map[string]temporal.VersioningIntent{
	"unspecified":          temporal.VersioningIntentUnspecified,
	"compatible":           temporal.VersioningIntentCompatible,
	"default":              temporal.VersioningIntentDefault,
	"inherit_build_id":     temporal.VersioningIntentInheritBuildID,
	"inheritbuildid":       temporal.VersioningIntentInheritBuildID,
	"use_assignment_rules": temporal.VersioningIntentUseAssignmentRules,
	"useassignmentrules":   temporal.VersioningIntentUseAssignmentRules,
}

func parseVersioningIntent(val string) (temporal.VersioningIntent, error) {
	key := strings.ToLower(val)
	if intent, ok := versioningIntentMap[key]; ok {
		return intent, nil
	}
	return temporal.VersioningIntentUnspecified, fmt.Errorf("invalid versioning intent '%s'", val)
}

func durationToTime(d *model.Duration) (time.Duration, error) {
	if d == nil {
		return 0, nil
	}

	if inline := d.AsInline(); inline != nil {
		return utils.ToDuration(d), nil
	}

	expr := d.AsExpression()
	if expr == "" {
		return 0, nil
	}

	return parseISODuration(expr)
}

var iso8601DurationRe = regexp.MustCompile(`^P(?:(\d+)Y)?(?:(\d+)M)?(?:(\d+)D)?(?:T(?:(\d+)H)?(?:(\d+)M)?(?:(\d+(?:\.\d+)?)S)?)?$`)

func parseISODuration(expr string) (time.Duration, error) {
	matches := iso8601DurationRe.FindStringSubmatch(expr)
	if len(matches) == 0 {
		return 0, fmt.Errorf("invalid ISO8601 duration: %s", expr)
	}

	if matches[1] != "" || matches[2] != "" {
		return 0, fmt.Errorf("ISO8601 years/months not supported: %s", expr)
	}

	days, err := toDurationComponent(matches[3], 24*time.Hour)
	if err != nil {
		return 0, err
	}
	hours, err := toDurationComponent(matches[4], time.Hour)
	if err != nil {
		return 0, err
	}
	minutes, err := toDurationComponent(matches[5], time.Minute)
	if err != nil {
		return 0, err
	}
	seconds, err := parseSeconds(matches[6])
	if err != nil {
		return 0, err
	}

	return days + hours + minutes + seconds, nil
}

func toDurationComponent(value string, unit time.Duration) (time.Duration, error) {
	if value == "" {
		return 0, nil
	}
	v, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid ISO8601 duration component '%s': %w", value, err)
	}
	return time.Duration(v) * unit, nil
}

func parseSeconds(value string) (time.Duration, error) {
	if value == "" {
		return 0, nil
	}
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid ISO8601 seconds '%s': %w", value, err)
	}
	return time.Duration(math.Round(f * float64(time.Second))), nil
}
