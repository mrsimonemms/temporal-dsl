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
	"testing"
	"time"

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/stretchr/testify/require"
)

func TestDurationToTime(t *testing.T) {
	t.Run("inline", func(t *testing.T) {
		d := &model.Duration{Value: model.DurationInline{Seconds: 5}}
		got, err := durationToTime(d)
		require.NoError(t, err)
		require.Equal(t, 5*time.Second, got)
	})

	t.Run("expression", func(t *testing.T) {
		d := &model.Duration{Value: model.DurationExpression{Expression: "PT1H30M"}}
		got, err := durationToTime(d)
		require.NoError(t, err)
		require.Equal(t, time.Hour+30*time.Minute, got)
	})

	t.Run("unsupported", func(t *testing.T) {
		d := &model.Duration{Value: model.DurationExpression{Expression: "P1Y"}}
		_, err := durationToTime(d)
		require.Error(t, err)
	})
}

func TestConvertRetryPolicy(t *testing.T) {
	maxAttempts := int32(3)
	retry, err := convertRetryPolicy(&ActivityRetryPolicy{
		InitialInterval: &model.Duration{Value: model.DurationInline{Seconds: 1}},
		BackoffCoefficient: func() *float64 {
			v := 2.0
			return &v
		}(),
		MaximumAttempts: &maxAttempts,
	})
	require.NoError(t, err)
	require.NotNil(t, retry)
	require.Equal(t, 3, int(retry.MaximumAttempts))
	require.Equal(t, 2.0, retry.BackoffCoefficient)
	require.Equal(t, time.Second, retry.InitialInterval)

	retry, err = convertRetryPolicy(&ActivityRetryPolicy{})
	require.NoError(t, err)
	require.Nil(t, retry)
}
