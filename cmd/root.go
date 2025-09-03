/*
Copyright © 2025 Simon Emms <simon@simonemms.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	gh "github.com/mrsimonemms/golang-helpers"
	"github.com/mrsimonemms/golang-helpers/temporal"
	"github.com/mrsimonemms/temporal-codec-server/packages/golang/algorithms/aes"
	"github.com/mrsimonemms/temporal-dsl/pkg/dsl"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

var rootOpts struct {
	ConvertData          bool
	ConvertKeyPath       string
	EnvPrefix            string
	FilePath             string
	HealthListenAddress  string
	LogLevel             string
	MetricsListenAddress string
	MetricsPrefix        string
	TaskQueue            string
	TemporalAddress      string
	TemporalAPIKey       string
	TemporalTLSEnabled   bool
	TemporalNamespace    string
	Validate             bool
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:           "temporal-dsl",
	Version:       Version,
	Short:         "Build Temporal workflows from YAML",
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		level, err := zerolog.ParseLevel(rootOpts.LogLevel)
		if err != nil {
			return err
		}
		zerolog.SetGlobalLevel(level)

		return nil
	},
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if rootOpts.EnvPrefix == "" {
			return gh.FatalError{
				Msg: "Env prefix cannot be empty",
				WithParams: func(l *zerolog.Event) *zerolog.Event {
					return l.Str("prefix", rootOpts.EnvPrefix)
				},
			}
		}
		if strings.HasSuffix(rootOpts.EnvPrefix, "_") {
			return gh.FatalError{
				Msg: "Env prefix cannot end with underscore (_)",
				WithParams: func(l *zerolog.Event) *zerolog.Event {
					return l.Str("prefix", rootOpts.EnvPrefix)
				},
			}
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		defer func() {
			if r := recover(); r != nil {
				log.Fatal().Interface("recover", r).Msg("Recovered from panic")
			}
		}()

		var converter converter.DataConverter
		if rootOpts.ConvertData {
			keys, err := aes.ReadKeyFile(rootOpts.ConvertKeyPath)
			if err != nil {
				return gh.FatalError{
					Cause: err,
					Msg:   "Unable to get keys from file",
					WithParams: func(l *zerolog.Event) *zerolog.Event {
						return l.Str("keypath", rootOpts.ConvertKeyPath)
					},
				}
			}
			converter = aes.DataConverter(keys)
		}

		// The client and worker are heavyweight objects that should be created once per process.
		log.Trace().Msg("Connecting to Temporal")
		c, err := temporal.NewConnection(
			temporal.WithHostPort(rootOpts.TemporalAddress),
			temporal.WithNamespace(rootOpts.TemporalNamespace),
			temporal.WithTLS(rootOpts.TemporalTLSEnabled),
			temporal.WithAPICredentials(rootOpts.TemporalAPIKey),
			temporal.WithDataConverter(converter),
			temporal.WithZerolog(&log.Logger),
			temporal.WithPrometheusMetrics(rootOpts.MetricsListenAddress, rootOpts.MetricsPrefix),
		)
		if err != nil {
			return gh.FatalError{
				Cause: err,
				Msg:   "Unable to create client",
			}
		}
		defer func() {
			log.Trace().Msg("Closing Temporal connection")
			c.Close()
			log.Trace().Msg("Temporal connection closed")
		}()

		log.Debug().Msg("Starting health check service")
		ctx := context.Background()
		temporal.NewHealthCheck(ctx, rootOpts.TaskQueue, rootOpts.HealthListenAddress, c)

		// Load the workflow file
		wf, err := dsl.LoadFromFile(rootOpts.FilePath, rootOpts.EnvPrefix)
		if err != nil {
			return gh.FatalError{
				Cause: err,
				Msg:   "Error loading workflow",
			}
		}

		if rootOpts.Validate {
			log.Debug().Msg("Running validation")
			if res, err := wf.Validate(); err != nil {
				return gh.FatalError{
					Cause: err,
					Msg:   "Error validating",
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

		log.Info().Msg("Upserting schedules")
		if err := dsl.UpsertSchedule(ctx, c, wf, rootOpts.TaskQueue); err != nil {
			return gh.FatalError{
				Cause: err,
				Msg:   "Error upserting Temporal schedules",
			}
		}

		log.Info().Str("deploymentName", wf.WorkflowName()).Str("buildID", wf.Document().Version).Msg("New versioned workflow")
		w := worker.New(c, rootOpts.TaskQueue, worker.Options{
			DeploymentOptions: worker.DeploymentOptions{
				UseVersioning: true,
				Version: worker.WorkerDeploymentVersion{
					DeploymentName: wf.WorkflowName(),
					BuildId:        wf.Document().Version,
				},
				DefaultVersioningBehavior: workflow.VersioningBehaviorAutoUpgrade,
			},
		})

		workflows, err := wf.BuildWorkflows()
		if err != nil {
			return gh.FatalError{
				Cause: err,
				Msg:   "Error building workflows",
			}
		}

		for _, wf := range workflows {
			log.Debug().Str("name", wf.Name).Msg("Registering workflow")
			w.RegisterWorkflowWithOptions(wf.Workflow, workflow.RegisterOptions{
				Name: wf.Name,
			})
		}

		log.Debug().Msg("Registering activities")
		w.RegisterActivity(wf.Activities())

		err = w.Run(worker.InterruptCh())
		if err != nil {
			return gh.FatalError{
				Cause: err,
				Msg:   "Unable to start worker",
			}
		}

		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(gh.HandleFatalError(err))
	}
}

func init() {
	viper.AutomaticEnv()

	rootCmd.Flags().BoolVar(
		&rootOpts.ConvertData, "convert-data",
		viper.GetBool("convert_data"), "Enable AES data conversion",
	)

	viper.SetDefault("converter_key_path", "keys.yaml")
	rootCmd.Flags().StringVar(
		&rootOpts.ConvertKeyPath, "converter-key-path",
		viper.GetString("converter_key_path"), "Path to AES conversion keys",
	)

	rootCmd.Flags().StringVarP(
		&rootOpts.FilePath, "file", "f",
		viper.GetString("workflow_file"), "Path to workflow file",
	)

	viper.SetDefault("env_prefix", "TDSL")
	rootCmd.Flags().StringVar(
		&rootOpts.EnvPrefix, "env-prefix",
		viper.GetString("env_prefix"), "Load envvars with this prefix to the workflow",
	)

	viper.SetDefault("health_listen_address", "0.0.0.0:3000")
	rootCmd.Flags().StringVar(
		&rootOpts.HealthListenAddress, "health-listen-address",
		viper.GetString("health_listen_address"), "Address of health server",
	)

	viper.SetDefault("log_level", zerolog.InfoLevel.String())
	rootCmd.PersistentFlags().StringVarP(
		&rootOpts.LogLevel, "log-level", "l",
		viper.GetString("log_level"), fmt.Sprintf("log level: %s", "Set log level"),
	)

	viper.SetDefault("metrics_listen_address", "0.0.0.0:9090")
	rootCmd.Flags().StringVar(
		&rootOpts.MetricsListenAddress, "metrics-listen-address",
		viper.GetString("metrics_listen_address"), "Address of Prometheus metrics server",
	)

	rootCmd.Flags().StringVar(
		&rootOpts.MetricsPrefix, "metrics-prefix",
		viper.GetString("metrics_prefix"), "Prefix for metrics",
	)

	viper.SetDefault("task_queue", "temporal-dsl")
	rootCmd.Flags().StringVarP(
		&rootOpts.TaskQueue, "task-queue", "q",
		viper.GetString("task_queue"), "Task queue name",
	)

	viper.SetDefault("temporal_address", client.DefaultHostPort)
	rootCmd.Flags().StringVarP(
		&rootOpts.TemporalAddress, "temporal-address", "H",
		viper.GetString("temporal_address"), "Address of the Temporal server",
	)

	rootCmd.Flags().StringVar(
		&rootOpts.TemporalAPIKey, "temporal-api-key",
		viper.GetString("temporal_api_key"), "API key for Temporal authentication",
	)
	// Hide the default value to avoid spaffing the API to command line
	apiKey := rootCmd.Flags().Lookup("temporal-api-key")
	if s := apiKey.Value; s.String() != "" {
		apiKey.DefValue = "***"
	}

	viper.SetDefault("temporal_namespace", client.DefaultNamespace)
	rootCmd.Flags().StringVarP(
		&rootOpts.TemporalNamespace, "temporal-namespace", "n",
		viper.GetString("temporal_namespace"), "Temporal namespace to use",
	)

	viper.SetDefault("temporal_tls", client.DefaultNamespace)
	rootCmd.Flags().BoolVar(
		&rootOpts.TemporalTLSEnabled, "temporal-tls",
		viper.GetBool("temporal_tls"), "Enable TLS Temporal connection",
	)

	viper.SetDefault("validate", true)
	rootCmd.Flags().BoolVar(
		&rootOpts.Validate, "validate",
		viper.GetBool("validate"), "Run workflow validation",
	)
}
