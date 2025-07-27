package cmd

import (
	"fmt"
	"os"

	"github.com/busybytelab.com/rag-cli/pkg/database"
	"github.com/busybytelab.com/rag-cli/pkg/output"
	"github.com/spf13/cobra"
)

var collectionCmd = &cobra.Command{
	Use:   "collection",
	Short: "Manage collections",
	Long: `Manage collections of documents for RAG operations.

Collections are groups of documents that are indexed together and can be searched
or used for chat sessions. Each collection can contain documents from multiple folders.

Examples:
  # List all collections
  rag-cli collection list

  # Create a new collection
  rag-cli collection create my-docs -d "My documentation collection" -f ./docs

  # Show collection details
  rag-cli collection show abc123

  # Edit collection details
  rag-cli collection edit abc123 --new-name "updated-name" --new-description "Updated description"

  # Add folder to collection
  rag-cli collection add-folder abc123 --folder ./new-docs

  # Remove folder from collection
  rag-cli collection remove-folder abc123 --folder ./old-docs

  # Delete a collection (with confirmation)
  rag-cli collection delete abc123 --force`,
}

var createCollectionCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new collection",
	Long: `Create a new collection with the specified name, description, and folders.

A collection groups documents from specified folders for indexing and searching.
The collection will be created immediately, but documents need to be indexed
separately using the 'index' command.

Examples:
  # Create a collection with a single folder
  rag-cli collection create my-docs -d "My documentation" -f ./docs

  # Create a collection with multiple folders
  rag-cli collection create project-docs -d "Project documentation" -f ./docs -f ./guides -f ./api`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		description, _ := cmd.Flags().GetString("description")
		folders, _ := cmd.Flags().GetStringSlice("folders")

		if len(folders) == 0 {
			return fmt.Errorf("at least one folder must be specified")
		}

		// Validate folders exist
		for _, folder := range folders {
			if _, err := os.Stat(folder); os.IsNotExist(err) {
				return fmt.Errorf("folder does not exist: %s", folder)
			}
		}

		// Connect to database
		db, err := database.NewConnection(&cfg.Database)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		// Create collection manager
		collectionMgr := database.NewCollectionManager(db)

		// Create collection
		collection, err := collectionMgr.CreateCollection(name, description, folders)
		if err != nil {
			return fmt.Errorf("failed to create collection: %w", err)
		}

		output.Success("Collection created successfully!")
		output.KeyValue("ID", collection.ID)
		output.KeyValue("Name", collection.Name)
		output.KeyValue("Description", collection.Description)
		output.KeyValuef("Folders", "%v", collection.Folders)

		return nil
	},
}

var listCollectionsCmd = &cobra.Command{
	Use:   "list",
	Short: "List all collections",
	Long: `List all collections with their details and statistics.

Shows all collections in the database along with their metadata,
folder paths, and document statistics.

Examples:
  # List all collections
  rag-cli collection list

  # List collections with verbose output
  rag-cli collection list -v`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Connect to database
		db, err := database.NewConnection(&cfg.Database)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		// Create collection manager
		collectionMgr := database.NewCollectionManager(db)

		// List collections
		collections, err := collectionMgr.ListCollections()
		if err != nil {
			return fmt.Errorf("failed to list collections: %w", err)
		}

		if len(collections) == 0 {
			output.Info("No collections found.")
			return nil
		}

		output.Bold("Collections:")
		for _, collection := range collections {
			output.Info("")
			output.KeyValue("ID", collection.ID)
			output.KeyValue("Name", collection.Name)
			output.KeyValue("Description", collection.Description)
			output.KeyValuef("Folders", "%v", collection.Folders)
			output.KeyValuef("Stats", "%d documents, %d chunks, %d bytes",
				collection.Stats.TotalDocuments,
				collection.Stats.TotalChunks,
				collection.Stats.TotalSize)
			output.KeyValue("Created", collection.CreatedAt.Format("2006-01-02 15:04:05"))
		}

		return nil
	},
}

var showCollectionCmd = &cobra.Command{
	Use:   "show [collection-id-or-name]",
	Short: "Show collection details",
	Long: `Show detailed information about a specific collection.

Displays comprehensive information about a collection including its metadata,
folder paths, document statistics, and timestamps.

Examples:
  # Show collection details by ID
  rag-cli collection show 550e8400-e29b-41d4-a716-446655440000

  # Show collection details by name
  rag-cli collection show my-docs-collection`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		// Connect to database
		db, err := database.NewConnection(&cfg.Database)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		// Create collection manager
		collectionMgr := database.NewCollectionManager(db)

		// Get collection by ID or name
		collection, err := collectionMgr.GetCollectionByIdOrName(id)
		if err != nil {
			return fmt.Errorf("failed to get collection: %w", err)
		}

		output.Bold("Collection Details:")
		output.KeyValue("ID", collection.ID)
		output.KeyValue("Name", collection.Name)
		output.KeyValue("Description", collection.Description)
		output.KeyValuef("Folders", "%v", collection.Folders)
		output.KeyValuef("Stats", "%d documents, %d chunks, %d bytes",
			collection.Stats.TotalDocuments,
			collection.Stats.TotalChunks,
			collection.Stats.TotalSize)
		output.KeyValue("Created", collection.CreatedAt.Format("2006-01-02 15:04:05"))
		output.KeyValue("Updated", collection.UpdatedAt.Format("2006-01-02 15:04:05"))

		return nil
	},
}

var deleteCollectionCmd = &cobra.Command{
	Use:   "delete [collection-id-or-name]",
	Short: "Delete a collection",
	Long: `Delete a collection and all its documents.

This operation is irreversible and will permanently delete the collection
and all its indexed documents. Use with caution.

Examples:
  # Delete a collection by ID (will prompt for confirmation)
  rag-cli collection delete 550e8400-e29b-41d4-a716-446655440000

  # Delete a collection by name (will prompt for confirmation)
  rag-cli collection delete my-docs-collection

  # Force delete without confirmation
  rag-cli collection delete my-docs-collection -f

  # Force delete using long flag
  rag-cli collection delete my-docs-collection --force`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		force, _ := cmd.Flags().GetBool("force")

		if !force {
			output.Warning("This will delete the collection and all its documents.")
			output.Info("Use --force to confirm.")
			return nil
		}

		// Connect to database
		db, err := database.NewConnection(&cfg.Database)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		// Create collection manager
		collectionMgr := database.NewCollectionManager(db)

		// Get collection by ID or name first to validate it exists
		collection, err := collectionMgr.GetCollectionByIdOrName(id)
		if err != nil {
			return fmt.Errorf("failed to get collection: %w", err)
		}

		// Delete collection using the actual ID
		err = collectionMgr.DeleteCollection(collection.ID)
		if err != nil {
			return fmt.Errorf("failed to delete collection: %w", err)
		}

		output.Success("Collection deleted successfully!")

		return nil
	},
}

var editCollectionCmd = &cobra.Command{
	Use:   "edit [collection-id-or-name]",
	Short: "Edit collection details",
	Long: `Edit a collection's name and description.

Updates the collection's metadata while preserving all documents and folders.
You can update either the name, description, or both. Fields not specified
will remain unchanged.

Examples:
  # Edit collection by ID (update both name and description)
  rag-cli collection edit 550e8400-e29b-41d4-a716-446655440000 --new-name "updated-name" --new-description "Updated description"

  # Edit collection by name (update both name and description)
  rag-cli collection edit my-docs-collection --new-name "new-name" --new-description "New description"

  # Update only the description (name remains unchanged)
  rag-cli collection edit my-docs-collection --new-description "Updated description"

  # Update only the name (description remains unchanged)
  rag-cli collection edit my-docs-collection --new-name "new-name"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		newName, _ := cmd.Flags().GetString("new-name")
		newDescription, _ := cmd.Flags().GetString("new-description")

		// Check if at least one flag was provided
		if !cmd.Flags().Changed("new-name") && !cmd.Flags().Changed("new-description") {
			return fmt.Errorf("at least one of --new-name or --new-description must be specified")
		}

		// Connect to database
		db, err := database.NewConnection(&cfg.Database)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		// Create collection manager
		collectionMgr := database.NewCollectionManager(db)

		// Get collection by ID or name first to validate it exists
		collection, err := collectionMgr.GetCollectionByIdOrName(id)
		if err != nil {
			return fmt.Errorf("failed to get collection: %w", err)
		}

		// Prepare update parameters - only pass non-nil values for fields that were changed
		var namePtr *string
		var descriptionPtr *string

		if cmd.Flags().Changed("new-name") {
			namePtr = &newName
		}
		if cmd.Flags().Changed("new-description") {
			descriptionPtr = &newDescription
		}

		// Update collection
		updatedCollection, err := collectionMgr.UpdateCollection(collection.ID, namePtr, descriptionPtr)
		if err != nil {
			return fmt.Errorf("failed to update collection: %w", err)
		}

		output.Success("Collection updated successfully!")
		output.KeyValue("ID", updatedCollection.ID)
		output.KeyValue("Name", updatedCollection.Name)
		output.KeyValue("Description", updatedCollection.Description)
		output.KeyValuef("Folders", "%v", updatedCollection.Folders)

		return nil
	},
}

var addFolderCmd = &cobra.Command{
	Use:   "add-folder [collection-id-or-name]",
	Short: "Add a folder to a collection",
	Long: `Add a folder to an existing collection.

The folder will be added to the collection's folder list. Documents in the folder
will need to be indexed separately using the 'index' command.

Examples:
  # Add folder to collection by ID
  rag-cli collection add-folder 550e8400-e29b-41d4-a716-446655440000 --folder ./new-docs

  # Add folder to collection by name
  rag-cli collection add-folder my-docs-collection --folder ./additional-docs

  # Add folder using long flag
  rag-cli collection add-folder my-docs-collection --folder ./new-folder`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		folder, _ := cmd.Flags().GetString("folder")

		if folder == "" {
			return fmt.Errorf("folder must be specified")
		}

		// Validate folder exists
		if _, err := os.Stat(folder); os.IsNotExist(err) {
			return fmt.Errorf("folder does not exist: %s", folder)
		}

		// Connect to database
		db, err := database.NewConnection(&cfg.Database)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		// Create collection manager
		collectionMgr := database.NewCollectionManager(db)

		// Get collection by ID or name first to validate it exists
		collection, err := collectionMgr.GetCollectionByIdOrName(id)
		if err != nil {
			return fmt.Errorf("failed to get collection: %w", err)
		}

		// Add folder to collection
		updatedCollection, err := collectionMgr.AddFolderToCollection(collection.ID, folder)
		if err != nil {
			return fmt.Errorf("failed to add folder to collection: %w", err)
		}

		output.Success("Folder added to collection successfully!")
		output.KeyValue("ID", updatedCollection.ID)
		output.KeyValue("Name", updatedCollection.Name)
		output.KeyValuef("Folders", "%v", updatedCollection.Folders)

		return nil
	},
}

var removeFolderCmd = &cobra.Command{
	Use:   "remove-folder [collection-id-or-name]",
	Short: "Remove a folder from a collection",
	Long: `Remove a folder from a collection and delete all associated documents.

This operation will:
1. Remove the folder from the collection's folder list
2. Delete all documents and embeddings from that folder
3. Update collection statistics

This operation is irreversible. Use with caution.

Examples:
  # Remove folder from collection by ID
  rag-cli collection remove-folder 550e8400-e29b-41d4-a716-446655440000 --folder ./old-docs

  # Remove folder from collection by name
  rag-cli collection remove-folder my-docs-collection --folder ./deprecated-docs

  # Remove folder using long flag
  rag-cli collection remove-folder my-docs-collection --folder ./unused-folder`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		folder, _ := cmd.Flags().GetString("folder")

		if folder == "" {
			return fmt.Errorf("folder must be specified")
		}

		// Connect to database
		db, err := database.NewConnection(&cfg.Database)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		// Create collection manager
		collectionMgr := database.NewCollectionManager(db)

		// Get collection by ID or name first to validate it exists
		collection, err := collectionMgr.GetCollectionByIdOrName(id)
		if err != nil {
			return fmt.Errorf("failed to get collection: %w", err)
		}

		// Remove folder from collection
		updatedCollection, err := collectionMgr.RemoveFolderFromCollection(collection.ID, folder)
		if err != nil {
			return fmt.Errorf("failed to remove folder from collection: %w", err)
		}

		output.Success("Folder removed from collection successfully!")
		output.KeyValue("ID", updatedCollection.ID)
		output.KeyValue("Name", updatedCollection.Name)
		output.KeyValuef("Folders", "%v", updatedCollection.Folders)
		output.KeyValuef("Stats", "%d documents, %d chunks, %d bytes",
			updatedCollection.Stats.TotalDocuments,
			updatedCollection.Stats.TotalChunks,
			updatedCollection.Stats.TotalSize)

		return nil
	},
}

func init() {
	// Create collection flags
	createCollectionCmd.Flags().StringP("description", "d", "", "Collection description")
	createCollectionCmd.Flags().StringSliceP("folders", "f", []string{}, "Folders to include in collection")
	createCollectionCmd.MarkFlagRequired("folders")

	// Delete collection flags
	deleteCollectionCmd.Flags().BoolP("force", "f", false, "Force deletion without confirmation")

	// Edit collection flags
	editCollectionCmd.Flags().String("new-name", "", "New name for the collection")
	editCollectionCmd.Flags().String("new-description", "", "New description for the collection")

	// Add folder flags
	addFolderCmd.Flags().StringP("folder", "f", "", "Folder to add to collection")
	addFolderCmd.MarkFlagRequired("folder")

	// Remove folder flags
	removeFolderCmd.Flags().StringP("folder", "f", "", "Folder to remove from collection")
	removeFolderCmd.MarkFlagRequired("folder")

	// Add subcommands
	collectionCmd.AddCommand(createCollectionCmd)
	collectionCmd.AddCommand(listCollectionsCmd)
	collectionCmd.AddCommand(showCollectionCmd)
	collectionCmd.AddCommand(editCollectionCmd)
	collectionCmd.AddCommand(addFolderCmd)
	collectionCmd.AddCommand(removeFolderCmd)
	collectionCmd.AddCommand(deleteCollectionCmd)

	// Add to root
	rootCmd.AddCommand(collectionCmd)
}
