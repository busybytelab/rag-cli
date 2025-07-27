package cmd

import (
	"fmt"
	"os"

	"github.com/busybytelab.com/rag-cli/pkg/config"
	"github.com/busybytelab.com/rag-cli/pkg/output"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile    string
	configName string
	cfg        *config.Config
	noColor    bool
	verbose    bool
)

// GetConfig returns the current configuration
func GetConfig() *config.Config {
	return cfg
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "rag-cli",
	Short: "A CLI tool for RAG (Retrieval-Augmented Generation) with Ollama and PostgreSQL",
	Long: `rag-cli is a command-line interface for building and querying RAG systems.
It allows you to create collections from various sources, index documents with embeddings,
and perform vector search and chat with your documents using Ollama and PostgreSQL.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Handle color flag
		if noColor {
			output.DisableColors()
		}

		// Set the global configuration name
		config.CurrentConfigName = configName

		var err error
		cfg, err = config.LoadConfig(configName)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Override config with command line flags if provided
		if cmd.Flags().Changed("ollama-host") {
			host, _ := cmd.Flags().GetString("ollama-host")
			cfg.Ollama.Host = host
		}
		if cmd.Flags().Changed("ollama-port") {
			port, _ := cmd.Flags().GetInt("ollama-port")
			cfg.Ollama.Port = port
		}

		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.rag-cli/config.yaml)")
	rootCmd.PersistentFlags().StringVarP(&configName, "config-name", "c", "", "config name to use (e.g. 'dev' for $HOME/.rag-cli/dev.yaml)")

	// Ollama flags
	rootCmd.PersistentFlags().String("ollama-host", "", "Ollama server host (default is localhost)")
	rootCmd.PersistentFlags().Int("ollama-port", 0, "Ollama server port (default is 11434)")

	// Output flags
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable color output")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".rag-cli" (without extension).
		viper.AddConfigPath(home + "/.rag-cli")
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		if verbose {
			fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		}
	}
}
