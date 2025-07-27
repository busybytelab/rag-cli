package cmd

import (
	"fmt"

	"github.com/busybytelab.com/rag-cli/pkg/database"
	"github.com/busybytelab.com/rag-cli/pkg/output"
	"github.com/spf13/cobra"
)

var documentsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Manage documents",
	Long: `Manage documents in collections.

Documents are the indexed content from files in collection folders.
You can list documents by collection and folder, and view individual document chunks.

Examples:
  # List documents in a collection folder
  rag-cli docs list --collection my-docs-collection --folder ./docs

  # List documents with file filter
  rag-cli docs list --collection my-docs-collection --folder ./docs --filter "*.md"

  # Show document chunk content
  rag-cli docs show --id 550e8400-e29b-41d4-a716-446655440000

  # Show document chunk content by collection and file path
  rag-cli docs show --collection my-docs-collection --file ./docs/README.md

  # Remove document chunk
  rag-cli docs remove --id 550e8400-e29b-41d4-a716-446655440000`,
}

var listDocumentsCmd = &cobra.Command{
	Use:   "list",
	Short: "List documents in a collection folder",
	Long: `List documents from a specific folder in a collection.

Shows all documents in the specified folder with their metadata, sorted by file path.
Supports pagination with limit and offset parameters, and file pattern filtering.

Examples:
  # List documents in a folder (default limit: 50)
  rag-cli docs list --collection my-docs-collection --folder ./docs

  # List documents with custom limit
  rag-cli docs list --collection my-docs-collection --folder ./docs --limit 100

  # List documents with pagination
  rag-cli docs list --collection my-docs-collection --folder ./docs --limit 20 --offset 40

  # List documents by collection ID
  rag-cli docs list --collection 550e8400-e29b-41d4-a716-446655440000 --folder ./docs

  # Filter documents by file pattern (Markdown files only)
  rag-cli docs list --collection my-docs-collection --folder ./docs --filter "*.md"

  # Filter documents by file pattern (Go files containing "coll")
  rag-cli docs list --collection my-docs-collection --folder ./docs --filter "*coll*.go"

  # Filter documents by file pattern (all text files)
  rag-cli docs list --collection my-docs-collection --folder ./docs --filter "*.txt"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		collectionID, _ := cmd.Flags().GetString("collection")
		folder, _ := cmd.Flags().GetString("folder")
		fileFilter, _ := cmd.Flags().GetString("filter")
		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")

		if collectionID == "" {
			return fmt.Errorf("collection must be specified")
		}
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
		collection, err := collectionMgr.GetCollectionByIdOrName(collectionID)
		if err != nil {
			return fmt.Errorf("failed to get collection: %w", err)
		}

		// Validate that the folder exists in the collection
		folderExists := false
		for _, existingFolder := range collection.Folders {
			if existingFolder == folder {
				folderExists = true
				break
			}
		}

		if !folderExists {
			return fmt.Errorf("folder '%s' does not exist in collection '%s'", folder, collection.Name)
		}

		// Create document manager
		documentMgr := database.NewDocumentManager(db)

		// List documents in the folder
		var documents []*database.Document
		if fileFilter != "" {
			documents, err = documentMgr.ListDocumentsByFolderWithFilter(collection.ID, folder, fileFilter, limit, offset)
		} else {
			documents, err = documentMgr.ListDocumentsByFolder(collection.ID, folder, limit, offset)
		}
		if err != nil {
			return fmt.Errorf("failed to list documents: %w", err)
		}

		if len(documents) == 0 {
			if fileFilter != "" {
				output.Info("No documents found in folder '%s' matching filter '%s'", folder, fileFilter)
			} else {
				output.Info("No documents found in folder '%s'", folder)
			}
			return nil
		}

		output.Bold("Documents in folder '%s':", folder)
		if fileFilter != "" {
			output.Info("Filter: %s", fileFilter)
		}
		output.Info("")

		for i, doc := range documents {
			output.Info("Document %d:", i+1)
			output.KeyValue("ID", doc.ID)
			output.KeyValue("File Path", doc.FilePath)
			output.KeyValue("File Name", doc.FileName)
			output.KeyValuef("Chunk Index", "%d", doc.ChunkIndex)
			output.KeyValuef("Content Length", "%d", len(doc.Content))
			output.KeyValue("Created", doc.CreatedAt.Format("2006-01-02 15:04:05"))
			output.KeyValue("Updated", doc.UpdatedAt.Format("2006-01-02 15:04:05"))

			if i < len(documents)-1 {
				output.Info("")
			}
		}

		output.Info("")
		output.KeyValuef("Total Documents", "%d", len(documents))
		output.KeyValuef("Limit", "%d", limit)
		output.KeyValuef("Offset", "%d", offset)

		return nil
	},
}

var showDocumentCmd = &cobra.Command{
	Use:   "show",
	Short: "Show document chunk content",
	Long: `Show the content of a specific document chunk.

You can specify the document either by its ID directly, or by collection and file path.
Each document represents a chunk of the original file.

Examples:
  # Show document chunk by ID
  rag-cli docs show --id 550e8400-e29b-41d4-a716-446655440000

  # Show document chunk by collection and file path
  rag-cli docs show --collection my-docs-collection --file ./docs/README.md

  # Show document chunk by collection ID and file path
  rag-cli docs show --collection 550e8400-e29b-41d4-a716-446655440000 --file ./docs/README.md

  # Show document chunk with shorthand flags
  rag-cli docs show --collection my-docs-collection -f ./docs/README.md`,
	RunE: func(cmd *cobra.Command, args []string) error {
		documentID, _ := cmd.Flags().GetString("id")
		collectionID, _ := cmd.Flags().GetString("collection")
		filePath, _ := cmd.Flags().GetString("file")

		// Validate input parameters
		if documentID == "" && (collectionID == "" || filePath == "") {
			return fmt.Errorf("either --id or both --collection and --file must be specified")
		}
		if documentID != "" && (collectionID != "" || filePath != "") {
			return fmt.Errorf("cannot specify both --id and --collection/--file")
		}

		// Connect to database
		db, err := database.NewConnection(&cfg.Database)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		// Create document manager
		documentMgr := database.NewDocumentManager(db)

		var document *database.Document

		if documentID != "" {
			// Get document by ID
			document, err = documentMgr.GetDocumentByID(documentID)
			if err != nil {
				return fmt.Errorf("failed to get document: %w", err)
			}
		} else {
			// Get collection first
			collectionMgr := database.NewCollectionManager(db)
			collection, err := collectionMgr.GetCollectionByIdOrName(collectionID)
			if err != nil {
				return fmt.Errorf("failed to get collection: %w", err)
			}

			// Get document by collection ID and file path (first chunk)
			document, err = documentMgr.GetDocumentByPathAndIndex(collection.ID, filePath, 0)
			if err != nil {
				return fmt.Errorf("failed to get document: %w", err)
			}
		}

		// Display document information
		output.Bold("Document Details:")
		output.KeyValue("ID", document.ID)
		output.KeyValue("File Path", document.FilePath)
		output.KeyValue("File Name", document.FileName)
		output.KeyValuef("Chunk Index", "%d", document.ChunkIndex)
		output.KeyValuef("Content Length", "%d", len(document.Content))
		output.KeyValue("Created", document.CreatedAt.Format("2006-01-02 15:04:05"))
		output.KeyValue("Updated", document.UpdatedAt.Format("2006-01-02 15:04:05"))

		output.Info("")
		output.Bold("Content:")
		output.Info(document.Content)

		return nil
	},
}

var removeDocumentCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a document chunk",
	Long: `Remove a specific document chunk from a collection.

This operation will permanently delete the document chunk and its associated embedding.
Use with caution as this operation is irreversible.

Examples:
  # Remove document chunk by ID
  rag-cli docs remove --id 550e8400-e29b-41d4-a716-446655440000`,
	RunE: func(cmd *cobra.Command, args []string) error {
		documentID, _ := cmd.Flags().GetString("id")

		if documentID == "" {
			return fmt.Errorf("document ID must be specified")
		}

		// Connect to database
		db, err := database.NewConnection(&cfg.Database)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		// Create document manager
		documentMgr := database.NewDocumentManager(db)

		// Get document first to validate it exists and show details
		document, err := documentMgr.GetDocumentByID(documentID)
		if err != nil {
			return fmt.Errorf("failed to get document: %w", err)
		}

		// Delete document
		err = documentMgr.DeleteDocumentByID(documentID)
		if err != nil {
			return fmt.Errorf("failed to delete document: %w", err)
		}

		output.Success("Document chunk deleted successfully!")
		output.KeyValue("ID", document.ID)
		output.KeyValue("File Path", document.FilePath)
		output.KeyValue("File Name", document.FileName)
		output.KeyValuef("Chunk Index", "%d", document.ChunkIndex)

		return nil
	},
}

func init() {
	// List documents flags
	listDocumentsCmd.Flags().String("collection", "", "Collection ID or name")
	listDocumentsCmd.Flags().StringP("folder", "f", "", "Folder to list documents from")
	listDocumentsCmd.Flags().String("filter", "", "File pattern filter (e.g., '*.md', '*coll*.go')")
	listDocumentsCmd.Flags().IntP("limit", "l", 50, "Maximum number of documents to return")
	listDocumentsCmd.Flags().IntP("offset", "o", 0, "Number of documents to skip")
	listDocumentsCmd.MarkFlagRequired("collection")
	listDocumentsCmd.MarkFlagRequired("folder")

	// Show document flags
	showDocumentCmd.Flags().String("id", "", "Document ID")
	showDocumentCmd.Flags().String("collection", "", "Collection ID or name")
	showDocumentCmd.Flags().StringP("file", "f", "", "File path within the collection")

	// Remove document flags
	removeDocumentCmd.Flags().String("id", "", "Document ID")
	removeDocumentCmd.MarkFlagRequired("id")

	// Add subcommands
	documentsCmd.AddCommand(listDocumentsCmd)
	documentsCmd.AddCommand(showDocumentCmd)
	documentsCmd.AddCommand(removeDocumentCmd)

	// Add to root
	rootCmd.AddCommand(documentsCmd)
}
