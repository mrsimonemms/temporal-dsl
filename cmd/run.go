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

package cmd

import (
	"context"
	"fmt"

	gh "github.com/mrsimonemms/golang-helpers"
	"github.com/mrsimonemms/golang-helpers/temporal"
	"github.com/mrsimonemms/temporal-codec-server/packages/golang/algorithms/aes"
	"github.com/mrsimonemms/temporal-dsl/pkg/dsl"
	"github.com/mrsimonemms/temporal-dsl/pkg/utils"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/worker"
)

var runOpts struct {
	ConvertData          bool
	ConvertKeyPath       string
	EnvPrefix            string
	FilePath             string
	HealthListenAddress  string
	MetricsListenAddress string
	MetricsPrefix        string
	TemporalAddress      string
	TemporalAPIKey       string
	TemporalMTLSCertPath string
	TemporalMTLSKeyPath  string
	TemporalTLSEnabled   bool
	TemporalNamespace    string
	Validate             bool
}

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a worker implementing the DSL",
	RunE: func(cmd *cobra.Command, args []string) error {
		defer func() {
			if r := recover(); r != nil {
				var panicMsg string
				switch v := r.(type) {
				case error:
					panicMsg = v.Error()
				case string:
					panicMsg = v
				default:
					panicMsg = fmt.Sprintf("%+v", v)
				}

				log.Fatal().
					Str("type", fmt.Sprintf("%T", r)).
					Str("panicMsg", panicMsg).
					Msg("Recovered from panic")
			}
		}()

		workflowDefinition, err := dsl.LoadFromFile(runOpts.FilePath)
		if err != nil {
			log.Fatal().Err(err).Msg("Unable to load workflow file")
		}

		if runOpts.Validate {
			log.Debug().Msg("Running validation")

			validator, err := utils.NewValidator()
			if err != nil {
				log.Fatal().Err(err).Msg("Error creating validator")
			}

			if res, err := validator.ValidateStruct(workflowDefinition); err != nil {
				return gh.FatalError{
					Cause: err,
					Msg:   "Error creating validation stack",
				}
			} else if res != nil {
				return gh.FatalError{
					Cause: err,
					Msg:   "Validation failed",
					WithParams: func(l *zerolog.Event) *zerolog.Event {
						return l.Interface("validationErrors", res)
					},
				}
			}
			log.Debug().Msg("Validation passed")
		}

		var converter converter.DataConverter
		if runOpts.ConvertData {
			keys, err := aes.ReadKeyFile(runOpts.ConvertKeyPath)
			if err != nil {
				return gh.FatalError{
					Cause: err,
					Msg:   "Unable to get keys from file",
					WithParams: func(l *zerolog.Event) *zerolog.Event {
						return l.Str("keypath", runOpts.ConvertKeyPath)
					},
				}
			}
			converter = aes.DataConverter(keys)
		}

		// The client and worker are heavyweight objects that should be created once per process.
		log.Trace().Msg("Connecting to Temporal")
		client, err := temporal.NewConnection(
			temporal.WithHostPort(runOpts.TemporalAddress),
			temporal.WithNamespace(runOpts.TemporalNamespace),
			temporal.WithTLS(runOpts.TemporalTLSEnabled),
			temporal.WithAuthDetection(
				runOpts.TemporalAPIKey,
				runOpts.TemporalMTLSCertPath,
				runOpts.TemporalMTLSKeyPath,
			),
			temporal.WithDataConverter(converter),
			temporal.WithZerolog(&log.Logger),
			temporal.WithPrometheusMetrics(runOpts.MetricsListenAddress, runOpts.MetricsPrefix),
		)
		if err != nil {
			return gh.FatalError{
				Cause: err,
				Msg:   "Unable to create client",
			}
		}
		defer func() {
			log.Trace().Msg("Closing Temporal connection")
			client.Close()
			log.Trace().Msg("Temporal connection closed")
		}()

		taskQueue := workflowDefinition.Document.Namespace

		// Add underscore to the prefix
		prefix := runOpts.EnvPrefix
		prefix += "_"

		log.Debug().Str("prefix", prefix).Msg("Loading envvars to state")
		envvars := utils.LoadEnvvars(prefix)

		ctx := context.Background()

		log.Debug().Msg("Starting health check service")
		temporal.NewHealthCheck(ctx, taskQueue, runOpts.HealthListenAddress, client)

		log.Info().Msg("Updating schedules")
		if err := dsl.UpdateSchedules(ctx, client, workflowDefinition, envvars); err != nil {
			return gh.FatalError{
				Cause: err,
				Msg:   "Error updating Temporal schedules",
			}
		}

		log.Info().Str("task-queue", taskQueue).Msg("Starting workflow")

		pollerAutoscaler := worker.NewPollerBehaviorAutoscaling(worker.PollerBehaviorAutoscalingOptions{})
		temporalWorker := worker.New(client, taskQueue, worker.Options{
			WorkflowTaskPollerBehavior: pollerAutoscaler,
			ActivityTaskPollerBehavior: pollerAutoscaler,
			NexusTaskPollerBehavior:    pollerAutoscaler,
		})

		if err := dsl.NewWorkflow(temporalWorker, workflowDefinition, envvars); err != nil {
			return gh.FatalError{
				Cause: err,
				Msg:   "Unable to build workflow from DSL",
			}
		}

		if err := temporalWorker.Run(worker.InterruptCh()); err != nil {
			return gh.FatalError{
				Cause: err,
				Msg:   "Unable to start worker",
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().BoolVar(
		&runOpts.ConvertData, "convert-data",
		viper.GetBool("convert_data"), "Enable AES data conversion",
	)

	viper.SetDefault("converter_key_path", "keys.yaml")
	runCmd.Flags().StringVar(
		&runOpts.ConvertKeyPath, "converter-key-path",
		viper.GetString("converter_key_path"), "Path to AES conversion keys",
	)

	runCmd.Flags().StringVarP(
		&runOpts.FilePath, "file", "f",
		viper.GetString("workflow_file"), "Path to workflow file",
	)

	viper.SetDefault("env_prefix", "TDSL")
	runCmd.Flags().StringVar(
		&runOpts.EnvPrefix, "env-prefix",
		viper.GetString("env_prefix"), "Load envvars with this prefix to the workflow",
	)

	viper.SetDefault("health_listen_address", "0.0.0.0:3000")
	runCmd.Flags().StringVar(
		&runOpts.HealthListenAddress, "health-listen-address",
		viper.GetString("health_listen_address"), "Address of health server",
	)

	viper.SetDefault("metrics_listen_address", "0.0.0.0:9090")
	runCmd.Flags().StringVar(
		&runOpts.MetricsListenAddress, "metrics-listen-address",
		viper.GetString("metrics_listen_address"), "Address of Prometheus metrics server",
	)

	runCmd.Flags().StringVar(
		&runOpts.MetricsPrefix, "metrics-prefix",
		viper.GetString("metrics_prefix"), "Prefix for metrics",
	)

	viper.SetDefault("temporal_address", client.DefaultHostPort)
	runCmd.Flags().StringVarP(
		&runOpts.TemporalAddress, "temporal-address", "H",
		viper.GetString("temporal_address"), "Address of the Temporal server",
	)

	runCmd.Flags().StringVar(
		&runOpts.TemporalAPIKey, "temporal-api-key",
		viper.GetString("temporal_api_key"), "API key for Temporal authentication",
	)
	// Hide the default value to avoid spaffing the API to command line
	apiKey := runCmd.Flags().Lookup("temporal-api-key")
	if s := apiKey.Value; s.String() != "" {
		apiKey.DefValue = "***"
	}

	runCmd.Flags().StringVar(
		&runOpts.TemporalMTLSCertPath, "tls-client-cert-path",
		viper.GetString("temporal_tls_client_cert_path"), "Path to mTLS client cert, usually ending in .pem",
	)

	runCmd.Flags().StringVar(
		&runOpts.TemporalMTLSKeyPath, "tls-client-key-path",
		viper.GetString("temporal_tls_client_key_path"), "Path to mTLS client key, usually ending in .key",
	)

	viper.SetDefault("temporal_namespace", client.DefaultNamespace)
	runCmd.Flags().StringVarP(
		&runOpts.TemporalNamespace, "temporal-namespace", "n",
		viper.GetString("temporal_namespace"), "Temporal namespace to use",
	)

	runCmd.Flags().BoolVar(
		&runOpts.TemporalTLSEnabled, "temporal-tls",
		viper.GetBool("temporal_tls"), "Enable TLS Temporal connection",
	)

	viper.SetDefault("validate", true)
	runCmd.Flags().BoolVar(
		&runOpts.Validate, "validate",
		viper.GetBool("validate"), "Run workflow validation",
	)
}
