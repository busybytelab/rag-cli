package config

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	_ "github.com/lib/pq"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

// CurrentConfigName holds the current configuration name
var CurrentConfigName string

// Config represents the application configuration
type Config struct {
	ChatBackend      string          `mapstructure:"chat_backend" yaml:"chat_backend"`           // "ollama" or "openai"
	EmbeddingBackend string          `mapstructure:"embedding_backend" yaml:"embedding_backend"` // "ollama" or "openai" (defaults to chat_backend if not specified)
	Ollama           OllamaConfig    `mapstructure:"ollama" yaml:"ollama"`
	OpenAI           OpenAIConfig    `mapstructure:"openai" yaml:"openai"`
	Database         DatabaseConfig  `mapstructure:"database" yaml:"database"`
	Embedding        EmbeddingConfig `mapstructure:"embedding" yaml:"embedding"`
	General          GeneralConfig   `mapstructure:"general" yaml:"general"`
}

// OllamaConfig represents Ollama server configuration
type OllamaConfig struct {
	Host           string `mapstructure:"host" yaml:"host"`
	Port           int    `mapstructure:"port" yaml:"port"`
	TLS            bool   `mapstructure:"tls" yaml:"tls"`
	ChatModel      string `mapstructure:"chat_model" yaml:"chat_model"`
	EmbeddingModel string `mapstructure:"embedding_model" yaml:"embedding_model"`
}

// OpenAIConfig represents OpenAI API configuration
type OpenAIConfig struct {
	APIKey         string `mapstructure:"api_key" yaml:"api_key"`
	BaseURL        string `mapstructure:"base_url" yaml:"base_url"` // For local servers like llama-server
	ChatModel      string `mapstructure:"chat_model" yaml:"chat_model"`
	EmbeddingModel string `mapstructure:"embedding_model" yaml:"embedding_model"`
}

// DatabaseConfig represents PostgreSQL database configuration
type DatabaseConfig struct {
	Host     string `mapstructure:"host" yaml:"host"`
	Port     int    `mapstructure:"port" yaml:"port"`
	Name     string `mapstructure:"name" yaml:"name"`
	User     string `mapstructure:"user" yaml:"user"`
	Password string `mapstructure:"password" yaml:"password"`
	SSLMode  string `mapstructure:"ssl_mode" yaml:"ssl_mode"`
}

// EmbeddingConfig represents embedding configuration
type EmbeddingConfig struct {
	ChunkSize           int     `mapstructure:"chunk_size" yaml:"chunk_size"`
	ChunkOverlap        int     `mapstructure:"chunk_overlap" yaml:"chunk_overlap"`
	SimilarityThreshold float64 `mapstructure:"similarity_threshold" yaml:"similarity_threshold"`
	MaxResults          int     `mapstructure:"max_results" yaml:"max_results"`
	Dimensions          int     `mapstructure:"dimensions" yaml:"dimensions"` // Embedding vector dimensions
}

// GeneralConfig represents general application configuration
type GeneralConfig struct {
	LogLevel string `mapstructure:"log_level" yaml:"log_level"`
	DataDir  string `mapstructure:"data_dir" yaml:"data_dir"`
}

// Validate checks if the embedding configuration is valid
func (c *EmbeddingConfig) Validate() error {
	if c.ChunkSize <= 0 {
		return fmt.Errorf("chunk size must be greater than 0")
	}
	if c.ChunkOverlap < 0 {
		return fmt.Errorf("chunk overlap cannot be negative")
	}
	if c.ChunkOverlap >= c.ChunkSize {
		return fmt.Errorf("chunk overlap must be less than chunk size")
	}
	if c.SimilarityThreshold < 0 || c.SimilarityThreshold > 1 {
		return fmt.Errorf("similarity threshold must be between 0 and 1")
	}
	if c.MaxResults <= 0 {
		return fmt.Errorf("max results must be greater than 0")
	}
	if c.Dimensions <= 0 {
		return fmt.Errorf("embedding dimensions must be greater than 0")
	}
	return nil
}

// Validate checks if the configuration is valid and can connect to the database
func (c *Config) Validate() error {
	// Validate chat backend selection
	if c.ChatBackend != "ollama" && c.ChatBackend != "openai" {
		return fmt.Errorf("invalid chat_backend: %s. Must be 'ollama' or 'openai'", c.ChatBackend)
	}

	// Set embedding backend to chat backend if not specified
	if c.EmbeddingBackend == "" {
		c.EmbeddingBackend = c.ChatBackend
	}

	// Validate embedding backend selection
	if c.EmbeddingBackend != "ollama" && c.EmbeddingBackend != "openai" {
		return fmt.Errorf("invalid embedding_backend: %s. Must be 'ollama' or 'openai'", c.EmbeddingBackend)
	}

	// Validate embedding configuration
	if err := c.Embedding.Validate(); err != nil {
		return fmt.Errorf("embedding configuration error: %w", err)
	}

	// Validate database configuration
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("database configuration error: %w", err)
	}

	// Validate chat backend-specific configuration
	switch c.ChatBackend {
	case "ollama":
		if err := c.Ollama.Validate(); err != nil {
			return fmt.Errorf("ollama configuration error: %w", err)
		}
	case "openai":
		if err := c.OpenAI.Validate(); err != nil {
			return fmt.Errorf("openai configuration error: %w", err)
		}
	}

	// Validate embedding backend-specific configuration
	switch c.EmbeddingBackend {
	case "ollama":
		if err := c.Ollama.Validate(); err != nil {
			return fmt.Errorf("ollama embedding configuration error: %w", err)
		}
	case "openai":
		if err := c.OpenAI.Validate(); err != nil {
			return fmt.Errorf("openai embedding configuration error: %w", err)
		}
	}

	return nil
}

// Validate checks if the database configuration is valid
func (c *DatabaseConfig) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("database host cannot be empty")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("database port must be between 1 and 65535")
	}
	if c.Name == "" {
		return fmt.Errorf("database name cannot be empty")
	}
	if c.User == "" {
		return fmt.Errorf("database user cannot be empty")
	}

	// Validate SSL mode
	validSSLModes := map[string]bool{
		"disable":     true,
		"allow":       true,
		"prefer":      true,
		"require":     true,
		"verify-ca":   true,
		"verify-full": true,
	}
	if !validSSLModes[c.SSLMode] {
		return fmt.Errorf("invalid SSL mode: %s. Valid modes are: disable, allow, prefer, require, verify-ca, verify-full", c.SSLMode)
	}

	return nil
}

// Validate checks if the Ollama configuration is valid
func (c *OllamaConfig) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("ollama host cannot be empty")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("ollama port must be between 1 and 65535")
	}
	if c.ChatModel == "" {
		return fmt.Errorf("ollama chat_model cannot be empty")
	}
	if c.EmbeddingModel == "" {
		return fmt.Errorf("ollama embed model cannot be empty")
	}

	return nil
}

// Validate checks if the OpenAI configuration is valid
func (c *OpenAIConfig) Validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("openai api key cannot be empty")
	}
	if c.ChatModel == "" {
		return fmt.Errorf("openai chat_model cannot be empty")
	}
	if c.EmbeddingModel == "" {
		return fmt.Errorf("openai embed model cannot be empty")
	}

	return nil
}

// GetServerURL returns the complete Ollama server URL
func (c *OllamaConfig) GetServerURL() string {
	protocol := "http"
	if c.TLS {
		protocol = "https"
	}

	host := c.Host
	if host == "" {
		host = "localhost"
	}

	port := c.Port
	if port == 0 {
		port = 11434
	}

	return fmt.Sprintf("%s://%s:%d", protocol, host, port)
}

// GetBaseURL returns the OpenAI base URL
func (c *OpenAIConfig) GetBaseURL() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return "https://api.openai.com/v1"
}

// GetDSN returns the PostgreSQL connection string
func (c *DatabaseConfig) GetDSN() string {
	host := c.Host
	if host == "" {
		host = "localhost"
	}

	port := c.Port
	if port == 0 {
		port = 5432
	}

	sslMode := c.SSLMode
	if sslMode == "" {
		sslMode = "prefer"
	}

	// Build the DSN with proper SSL handling
	dsn := fmt.Sprintf("host=%s port=%d dbname=%s user=%s", host, port, c.Name, c.User)

	// Add password if provided
	if c.Password != "" {
		dsn += fmt.Sprintf(" password=%s", c.Password)
	}

	// Add SSL mode
	dsn += fmt.Sprintf(" sslmode=%s", sslMode)

	return dsn
}

// TestDatabaseConnection tests if the database configuration can successfully connect
func (c *DatabaseConfig) TestDatabaseConnection() error {
	dsn := c.GetDSN()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}
	defer db.Close()

	// Test the connection with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	return nil
}

// TestOllamaConnection tests if the Ollama configuration can successfully connect
func (c *OllamaConfig) TestOllamaConnection() error {
	// Import the client package to test connection
	// We'll make a simple HTTP request to the Ollama server
	url := c.GetServerURL() + "/api/tags"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to Ollama server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Ollama server returned status code: %d", resp.StatusCode)
	}

	return nil
}

// TestOpenAIConnection tests if the OpenAI configuration can successfully connect
func (c *OpenAIConfig) TestOpenAIConnection() error {
	// We'll make a simple HTTP request to the OpenAI API
	url := c.GetBaseURL() + "/models"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to OpenAI API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("OpenAI API returned status code: %d", resp.StatusCode)
	}

	return nil
}

// LoadConfig loads configuration from file or creates default if not exists
func LoadConfig(configName string) (*Config, error) {
	home, err := homedir.Dir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".rag-cli")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Set default configuration
	config := getDefaultConfig()

	// Determine config file name
	var configFile string
	if configName != "" {
		configFile = filepath.Join(configDir, configName+".yaml")
	} else {
		configFile = filepath.Join(configDir, "config.yaml")
	}

	// Check if config file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// Create default config file
		if err := SaveConfig(config, configFile); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
	} else {
		// Load existing config
		viper.SetConfigFile(configFile)
		if err := viper.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		if err := viper.Unmarshal(config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %w", err)
		}
	}

	// Validate the configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// SaveConfig saves configuration to file
func SaveConfig(config *Config, configFile string) error {
	viper.SetConfigFile(configFile)

	// Set the configuration values
	viper.Set("chat_backend", config.ChatBackend)
	viper.Set("embedding_backend", config.EmbeddingBackend)
	viper.Set("ollama", config.Ollama)
	viper.Set("openai", config.OpenAI)
	viper.Set("database", config.Database)
	viper.Set("embedding", config.Embedding)
	viper.Set("general", config.General)

	return viper.WriteConfig()
}

// getDefaultConfig returns the default configuration
func getDefaultConfig() *Config {
	home, _ := homedir.Dir()

	return &Config{
		ChatBackend:      "ollama", // Default to Ollama
		EmbeddingBackend: "ollama", // Default to Ollama
		Ollama: OllamaConfig{
			Host:           "localhost",
			Port:           11434,
			TLS:            false,
			ChatModel:      "qwen3:4b",
			EmbeddingModel: "dengcao/Qwen3-Embedding-0.6B:Q8_0",
		},
		OpenAI: OpenAIConfig{
			APIKey:         "",
			BaseURL:        "",
			ChatModel:      "gpt-4",
			EmbeddingModel: "text-embedding-3-small",
		},
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			Name:     "rag_cli",
			User:     "postgres",
			Password: "",
			SSLMode:  "prefer",
		},
		Embedding: EmbeddingConfig{
			ChunkSize:           1000,
			ChunkOverlap:        200,
			SimilarityThreshold: 0.7,
			MaxResults:          10,
			Dimensions:          1024, // Default to 1024 for dengcao/Qwen3-Embedding-0.6B:Q8_0
		},
		General: GeneralConfig{
			LogLevel: "info",
			DataDir:  filepath.Join(home, ".rag-cli", "data"),
		},
	}
}
