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
	"crypto/tls"
	"fmt"
	"os"
	"strings"

	"github.com/mrsimonemms/golang-helpers/temporal"
	"github.com/mrsimonemms/temporal-codec-server/packages/golang/algorithms/aes"
	tw "github.com/mrsimonemms/temporal-dsl/pkg/workflow"
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
	ConvertData        bool
	ConvertKeyPath     string
	EnvPrefix          string
	FilePath           string
	LogLevel           string
	TaskQueue          string
	TemporalAddress    string
	TemporalAPIKey     string
	TemporalTLSEnabled bool
	TemporalNamespace  string
	Validate           bool
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "temporal-dsl",
	Version: Version,
	Short:   "Build Temporal workflows from YAML",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		level, err := zerolog.ParseLevel(rootOpts.LogLevel)
		if err != nil {
			return err
		}
		zerolog.SetGlobalLevel(level)

		return nil
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		if rootOpts.EnvPrefix == "" {
			log.Fatal().Str("prefix", rootOpts.EnvPrefix).Msg("Env prefix cannot be empty")
		}
		if strings.HasSuffix(rootOpts.EnvPrefix, "_") {
			log.Fatal().Str("prefix", rootOpts.EnvPrefix).Msg("Env prefix cannot end with underscore (_)")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		connectionOpts := client.ConnectionOptions{}
		if rootOpts.TemporalTLSEnabled {
			// Use new to avoid a golint false positive
			log.Debug().Msg("Enabling TLS connection")
			connectionOpts.TLS = new(tls.Config)
		}
		var creds client.Credentials
		if rootOpts.TemporalAPIKey != "" {
			log.Debug().Msg("Using API key for authentcation")
			creds = client.NewAPIKeyStaticCredentials(rootOpts.TemporalAPIKey)
		}

		var converter converter.DataConverter
		if rootOpts.ConvertData {
			keys, err := aes.ReadKeyFile(rootOpts.ConvertKeyPath)
			if err != nil {
				log.Fatal().Err(err).Str("keypath", rootOpts.ConvertKeyPath).Msg("Unable to get keys from file")
			}
			converter = aes.DataConverter(keys)
		}

		// The client and worker are heavyweight objects that should be created once per process.
		c, err := client.Dial(client.Options{
			ConnectionOptions: connectionOpts,
			Credentials:       creds,
			HostPort:          rootOpts.TemporalAddress,
			Namespace:         rootOpts.TemporalNamespace,
			DataConverter:     converter,
			Logger:            temporal.NewZerologHandler(&log.Logger),
		})
		if err != nil {
			log.Fatal().Err(err).Msg("Unable to create client")
		}
		defer c.Close()

		// Load the workflow file
		wf, err := tw.LoadFromFile(rootOpts.FilePath, rootOpts.EnvPrefix)
		if err != nil {
			log.Fatal().Err(err).Msg("Error loading workflow")
		}

		if rootOpts.Validate {
			log.Debug().Msg("Running validation")
			if err := wf.Validate(); err != nil {
				log.Fatal().Err(err).Msg("Failed validation")
			}
		}

		w := worker.New(c, rootOpts.TaskQueue, worker.Options{})

		workflows, err := wf.BuildWorkflows()
		if err != nil {
			log.Fatal().Err(err).Msg("Error building workflows")
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
			log.Fatal().Err(err).Msg("Unable to start worker")
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	viper.AutomaticEnv()

	rootCmd.Flags().BoolVar(
		&rootOpts.ConvertData,
		"convert-data",
		viper.GetBool("convert_data"),
		"Enable AES data conversion",
	)

	viper.SetDefault("converter_key_path", "keys.yaml")
	rootCmd.Flags().StringVar(
		&rootOpts.ConvertKeyPath,
		"converter-key-path",
		viper.GetString("converter_key_path"),
		"Path to AES conversion keys",
	)

	rootCmd.Flags().StringVarP(
		&rootOpts.FilePath,
		"file",
		"f",
		viper.GetString("workflow_file"),
		"Path to workflow file",
	)

	viper.SetDefault("env_prefix", "TDSL")
	rootCmd.Flags().StringVar(
		&rootOpts.EnvPrefix,
		"env-prefix",
		viper.GetString("env_prefix"),
		"Load envvars with this prefix to the workflow",
	)

	viper.SetDefault("log_level", zerolog.InfoLevel.String())
	rootCmd.PersistentFlags().StringVarP(
		&rootOpts.LogLevel,
		"log-level",
		"l",
		viper.GetString("log_level"),
		fmt.Sprintf("log level: %s", "Set log level"),
	)

	viper.SetDefault("task_queue", "temporal-dsl")
	rootCmd.Flags().StringVarP(
		&rootOpts.TaskQueue,
		"task-queue",
		"q",
		viper.GetString("task_queue"),
		"Task queue name",
	)

	viper.SetDefault("temporal_address", client.DefaultHostPort)
	rootCmd.Flags().StringVarP(
		&rootOpts.TemporalAddress,
		"temporal-address",
		"H",
		viper.GetString("temporal_address"),
		"Address of the Temporal server",
	)

	rootCmd.Flags().StringVar(
		&rootOpts.TemporalAPIKey,
		"temporal-api-key",
		viper.GetString("temporal_api_key"),
		"API key for Temporal authentication",
	)
	// Hide the default value to avoid spaffing the API to command line
	apiKey := rootCmd.Flags().Lookup("temporal-api-key")
	if s := apiKey.Value; s.String() != "" {
		apiKey.DefValue = "***"
	}

	viper.SetDefault("temporal_namespace", client.DefaultNamespace)
	rootCmd.Flags().StringVarP(
		&rootOpts.TemporalNamespace,
		"temporal-namespace",
		"n",
		viper.GetString("temporal_namespace"),
		"Temporal namespace to use",
	)

	viper.SetDefault("temporal_tls", client.DefaultNamespace)
	rootCmd.Flags().BoolVar(
		&rootOpts.TemporalTLSEnabled,
		"temporal-tls",
		viper.GetBool("temporal_tls"),
		"Enable TLS Temporal connection",
	)

	viper.SetDefault("validate", true)
	rootCmd.Flags().BoolVar(
		&rootOpts.Validate,
		"validate",
		viper.GetBool("validate"),
		"Run workflow validation",
	)
}
