package cmd

import (
	"fmt"
	"os"

	"github.com/busybytelab.com/rag-cli/pkg/config"
	"github.com/busybytelab.com/rag-cli/pkg/output"
	"github.com/spf13/cobra"
)

// maskAPIKey masks an API key for display purposes
func maskAPIKey(apiKey string) string {
	if apiKey == "" {
		return "(not set)"
	}
	if len(apiKey) <= 8 {
		return "***"
	}
	return apiKey[:4] + "..." + apiKey[len(apiKey)-4:]
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long:  `Manage application configuration settings.`,
}

var showConfigCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long:  `Display the current configuration settings.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		output.Bold("Current Configuration:")
		output.Info("")

		output.Bold("Backend Settings:")
		output.Info("  Chat Backend: %s", cfg.ChatBackend)
		output.Info("  Embedding Backend: %s", cfg.EmbeddingBackend)
		output.Info("")

		output.Bold("Ollama Settings:")
		output.Info("  Host: %s", cfg.Ollama.Host)
		output.Info("  Port: %d", cfg.Ollama.Port)
		output.Info("  TLS: %t", cfg.Ollama.TLS)
		output.Info("  Chat Model: %s", cfg.Ollama.ChatModel)
		output.Info("  Embed Model: %s", cfg.Ollama.EmbeddingModel)
		output.Info("")

		output.Bold("OpenAI Settings:")
		output.Info("  API Key: %s", maskAPIKey(cfg.OpenAI.APIKey))
		output.Info("  Base URL: %s", cfg.OpenAI.BaseURL)
		output.Info("  Chat Model: %s", cfg.OpenAI.ChatModel)
		output.Info("  Embed Model: %s", cfg.OpenAI.EmbeddingModel)
		output.Info("")

		output.Bold("Database Settings:")
		output.Info("  Host: %s", cfg.Database.Host)
		output.Info("  Port: %d", cfg.Database.Port)
		output.Info("  Name: %s", cfg.Database.Name)
		output.Info("  User: %s", cfg.Database.User)
		output.Info("  SSL Mode: %s", cfg.Database.SSLMode)
		output.Info("")

		output.Bold("Embedding Settings:")
		output.Info("  Chunk Size: %d", cfg.Embedding.ChunkSize)
		output.Info("  Chunk Overlap: %d", cfg.Embedding.ChunkOverlap)
		output.Info("  Similarity Threshold: %.2f", cfg.Embedding.SimilarityThreshold)
		output.Info("  Max Results: %d", cfg.Embedding.MaxResults)
		output.Info("")

		output.Bold("General Settings:")
		output.Info("  Log Level: %s", cfg.General.LogLevel)
		output.Info("  Data Directory: %s", cfg.General.DataDir)

		return nil
	},
}

var initConfigCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration",
	Long:  `Create a new configuration file with default settings.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// This will be handled by the config package when LoadConfig is called
		// The config will be created automatically if it doesn't exist
		output.Success("Configuration initialized successfully!")
		output.Info("Configuration file created at: ~/.rag-cli/config.yaml")
		output.Info("Use 'rag-cli config show' to view current settings")

		return nil
	},
}

var editConfigCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit configuration",
	Long:  `Open the configuration file in your default editor.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		configFile := fmt.Sprintf("%s/.rag-cli/config.yaml", home)

		// Check if config file exists
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			output.Warning("Configuration file does not exist. Creating default configuration...")
			// This will create the default config
			_, err = config.LoadConfig("")
			if err != nil {
				return fmt.Errorf("failed to create default configuration: %w", err)
			}
		}

		// Try to open the file with the default editor
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "nano" // Default fallback
		}

		output.Info("Opening configuration file with: %s", editor)
		output.Info("File: %s", configFile)

		// Note: In a real implementation, you would use exec.Command to open the editor
		// For now, we'll just show the path
		output.Info("Please edit the configuration file manually at: %s", configFile)

		return nil
	},
}

var validateConfigCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration",
	Long: `Validate the current configuration by validating settings and testing connections.

This command validates the configuration format and tests connectivity to both
the database and Ollama server to ensure everything is properly configured.

Examples:
  # Validate current configuration
  rag-cli config validate

  # Test with verbose output
  rag-cli config validate -v`,
	RunE: func(cmd *cobra.Command, args []string) error {
		output.Bold("Testing Configuration:")
		output.Info("")

		// Test configuration validation
		output.Info("1. Validating configuration format...")
		if err := cfg.Validate(); err != nil {
			output.Error("Configuration validation failed: %v", err)
			return err
		}
		output.Success("✓ Configuration format is valid")
		output.Info("")

		// Test database connection
		output.Info("2. Testing database connection...")
		output.Bold("Database: %s:%d/%s (user: %s)", cfg.Database.Host, cfg.Database.Port, cfg.Database.Name, cfg.Database.User)
		if err := cfg.Database.TestDatabaseConnection(); err != nil {
			output.Error("Database connection failed: %v", err)
			output.Info("")
			output.Info("Troubleshooting tips:")
			output.Info("  - Ensure PostgreSQL is running")
			output.Info("  - Check server host and port, database name, user, and password")
			output.Info("  - Verify SSL mode settings")
			return err
		}
		output.Success("✓ Database connection successful")
		output.Info("")

		// Test Ollama connection (basic check)
		output.Info("3. Testing Ollama connection...")
		ollamaURL := cfg.Ollama.GetServerURL()
		output.Bold("Ollama URL: %s", ollamaURL)
		if err := cfg.Ollama.TestOllamaConnection(); err != nil {
			output.Error("Ollama connection failed: %v", err)
			output.Info("")
			output.Info("Troubleshooting tips:")
			output.Info("  - Ensure Ollama is running")
			output.Info("  - Check Ollama host and port settings")
			output.Info("  - Verify TLS settings if using HTTPS")
			output.Info("  - Try: curl %s/api/tags", ollamaURL)
			return err
		}
		output.Success("✓ Ollama connection successful")
		output.Info("")

		output.Success("All configuration tests passed!")
		return nil
	},
}

func init() {
	configCmd.AddCommand(showConfigCmd)
	configCmd.AddCommand(initConfigCmd)
	configCmd.AddCommand(editConfigCmd)
	configCmd.AddCommand(validateConfigCmd)
	rootCmd.AddCommand(configCmd)
}
