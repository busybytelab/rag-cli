package cmd

import (
	"fmt"

	"github.com/busybytelab.com/rag-cli/pkg/database"
	"github.com/busybytelab.com/rag-cli/pkg/output"
	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Manage database migrations",
	Long: `Manage database schema migrations for RAG CLI.

This command helps you migrate your database schema when there are changes
to the embedding dimensions or other schema updates.

Examples:
  # Run all pending migrations
  rag-cli migrate up

  # Check current migration status
  rag-cli migrate status

  # Run migrations to a specific version
  rag-cli migrate up --to 2`,
}

var migrateUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Run pending migrations",
	Long: `Run all pending database migrations.

This will update your database schema to the latest version, including
any changes needed for embedding dimensions or other schema updates.

Examples:
  # Run all pending migrations
  rag-cli migrate up

  # Run migrations to a specific version
  rag-cli migrate up --to 2`,
	RunE: func(cmd *cobra.Command, args []string) error {
		toVersion, _ := cmd.Flags().GetInt("to")

		// Connect to database
		db, err := database.NewConnection(&cfg.Database)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		// Create database manager
		dbManager, err := database.NewDatabaseManager(&cfg.Database)
		if err != nil {
			return fmt.Errorf("failed to create database manager: %w", err)
		}
		defer dbManager.Close()

		// Get current version
		currentVersion, err := dbManager.GetMigrationVersion()
		if err != nil {
			return fmt.Errorf("failed to get current migration version: %w", err)
		}

		output.Info("Current migration version: %d", currentVersion)

		// Run migrations
		targetVersion := -1 // Run all migrations
		if toVersion > 0 {
			targetVersion = toVersion
		}

		if err := dbManager.RunMigrations(targetVersion); err != nil {
			return fmt.Errorf("failed to run migrations: %w", err)
		}

		// Get new version
		newVersion, err := dbManager.GetMigrationVersion()
		if err != nil {
			return fmt.Errorf("failed to get new migration version: %w", err)
		}

		if newVersion > currentVersion {
			output.Success("Migrations completed successfully!")
			output.KeyValue("Previous version", fmt.Sprintf("%d", currentVersion))
			output.KeyValue("Current version", fmt.Sprintf("%d", newVersion))
		} else {
			output.Info("Database is already up to date")
		}

		return nil
	},
}

var migrateStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show migration status",
	Long: `Show the current migration status of your database.

This displays the current migration version and available migrations.

Examples:
  # Show migration status
  rag-cli migrate status`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Connect to database
		db, err := database.NewConnection(&cfg.Database)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		// Create database manager
		dbManager, err := database.NewDatabaseManager(&cfg.Database)
		if err != nil {
			return fmt.Errorf("failed to create database manager: %w", err)
		}
		defer dbManager.Close()

		// Get current version
		currentVersion, err := dbManager.GetMigrationVersion()
		if err != nil {
			return fmt.Errorf("failed to get current migration version: %w", err)
		}

		output.Bold("Migration Status:")
		output.KeyValue("Current version", fmt.Sprintf("%d", currentVersion))
		output.KeyValue("Total migrations", fmt.Sprintf("%d", dbManager.GetTotalMigrations()))

		if currentVersion < dbManager.GetTotalMigrations() {
			output.Warning("Database is not up to date. Run 'rag-cli migrate up' to apply pending migrations.")
		} else {
			output.Success("Database is up to date")
		}

		return nil
	},
}

func init() {
	// Add flags
	migrateUpCmd.Flags().Int("to", 0, "Migrate to specific version (0 = run all)")

	// Add subcommands
	migrateCmd.AddCommand(migrateUpCmd)
	migrateCmd.AddCommand(migrateStatusCmd)

	// Add to root
	rootCmd.AddCommand(migrateCmd)
}
