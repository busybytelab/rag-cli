package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/busybytelab.com/rag-cli/pkg/client"
	"github.com/busybytelab.com/rag-cli/pkg/database"
	"github.com/busybytelab.com/rag-cli/pkg/embedding"
	"github.com/busybytelab.com/rag-cli/pkg/output"
	"github.com/spf13/cobra"
)

var indexCmd = &cobra.Command{
	Use:   "index [collection-id-or-name]",
	Short: "Index documents in a collection",
	Long: `Index documents from the folders specified in a collection.

This command processes all text files in the collection's folders, chunks them,
generates embeddings, and stores them in the database for searching.

Examples:
  # Index documents in a collection
  rag-cli index my-docs-collection

  # Force re-indexing (delete existing documents first)
  rag-cli index my-docs-collection --force

  # Force re-indexing using long flag
  rag-cli index my-docs-collection --force`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		collectionID := args[0]
		force, _ := cmd.Flags().GetBool("force")

		// Connect to database
		db, err := database.NewConnection(&cfg.Database)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		// Create managers
		collectionMgr := database.NewCollectionManager(db)
		documentMgr := database.NewDocumentManager(db)

		// Get collection by ID or name
		collection, err := collectionMgr.GetCollectionByIdOrName(collectionID)
		if err != nil {
			return fmt.Errorf("failed to get collection: %w", err)
		}

		output.KeyValue("Indexing collection", collection.Name)
		output.KeyValuef("Folders", "%v", collection.Folders)

		// Create embedder for generating embeddings
		embedder, err := client.NewEmbedder(cfg)
		if err != nil {
			return fmt.Errorf("failed to create embedder: %w", err)
		}

		// Create embedding service
		embeddingService := embedding.New(embedder, &cfg.Embedding)

		// Process each folder
		totalFiles := 0
		totalChunks := 0
		startTime := time.Now()

		for _, folder := range collection.Folders {
			output.Info("Processing folder: %s", folder)

			files, chunks, err := processFolder(folder, collection.ID, documentMgr, embeddingService, force)
			if err != nil {
				output.Error("Failed to process folder %s: %v", folder, err)
				continue
			}

			totalFiles += files
			totalChunks += chunks
		}

		// Update collection stats
		if err := collectionMgr.UpdateCollectionStats(collection.ID); err != nil {
			output.Warning("Failed to update collection stats: %v", err)
		}

		duration := time.Since(startTime)
		output.Success("Indexing completed!")
		output.KeyValuef("Total files processed", "%d", totalFiles)
		output.KeyValuef("Total chunks created", "%d", totalChunks)
		output.KeyValue("Duration", duration.String())

		return nil
	},
}

// processFolder processes all files in a folder
func processFolder(folderPath, collectionID string, documentMgr database.DocumentManager, embeddingService *embedding.Service, force bool) (int, int, error) {
	totalFiles := 0
	totalChunks := 0

	err := filepath.WalkDir(folderPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// Check if it's a text file
		if !isTextFile(path) {
			return nil
		}

		// Check if file is already indexed (unless force is true)
		if !force {
			// For now, we'll always re-index. In a production system, you'd check file modification time
			// and compare with the last indexed time in the database
		}

		output.Info("Processing file: %s", path)

		// Get file info for timestamps
		fileInfo, err := os.Stat(path)
		if err != nil {
			output.Error("Failed to get file info for %s: %v", path, err)
			return nil
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			output.Error("Failed to read file %s: %v", path, err)
			return nil // Continue with other files
		}

		// Delete existing documents for this file
		if err := documentMgr.DeleteDocumentsByPath(collectionID, path); err != nil {
			output.Error("Failed to delete existing documents for %s: %v", path, err)
			return nil
		}

		// Create metadata
		metadata := map[string]string{
			"file_path":     path,
			"file_name":     filepath.Base(path),
			"file_size":     fmt.Sprintf("%d", len(content)),
			"file_modified": fileInfo.ModTime().Format(time.RFC3339),
		}

		// Chunk the content
		chunks, err := embeddingService.ChunkText(string(content), metadata)
		if err != nil {
			output.Error("Failed to chunk file %s: %v", path, err)
			return nil
		}

		// Generate embeddings
		ctx := context.Background()
		if err := embeddingService.GenerateEmbeddings(ctx, chunks); err != nil {
			output.Error("Failed to generate embeddings for %s: %v", path, err)
			return nil
		}

		// Use file modification time for both created and updated timestamps
		// This represents when the file content was last changed
		fileTime := fileInfo.ModTime()

		// Store chunks in database
		for _, chunk := range chunks {
			metadataJSON, err := json.Marshal(chunk.Metadata)
			if err != nil {
				output.Error("Failed to marshal metadata: %v", err)
				continue
			}

			doc := &database.Document{
				CollectionID: collectionID,
				FilePath:     path,
				FileName:     filepath.Base(path),
				Content:      chunk.Content,
				ChunkIndex:   chunk.Index,
				Embedding:    chunk.Embedding,
				Metadata:     string(metadataJSON),
				CreatedAt:    fileTime, // Use file modification time as creation time
				UpdatedAt:    fileTime, // Use file modification time as update time
			}

			if err := documentMgr.InsertDocument(doc); err != nil {
				output.Error("Failed to insert document: %v", err)
				continue
			}
		}

		totalFiles++
		totalChunks += len(chunks)
		output.Info("Created %d chunks for %s", len(chunks), path)

		return nil
	})

	return totalFiles, totalChunks, err
}

// isTextFile checks if a file is a text file based on extension
func isTextFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	textExtensions := map[string]bool{
		".txt":  true,
		".md":   true,
		".rst":  true,
		".tex":  true,
		".log":  true,
		".csv":  true,
		".json": true,
		".xml":  true,
		".yaml": true,
		".yml":  true,
		".toml": true,
		".ini":  true,
		".cfg":  true,
		".conf": true,
		".sh":   true,
		".py":   true,
		".js":   true,
		".ts":   true,
		".go":   true,
		".rs":   true,
		".cpp":  true,
		".c":    true,
		".h":    true,
		".hpp":  true,
		".java": true,
		".cs":   true,
		".php":  true,
		".rb":   true,
		".pl":   true,
		".sql":  true,
		".html": true,
		".htm":  true,
		".css":  true,
		".scss": true,
		".sass": true,
		".less": true,
	}

	return textExtensions[ext]
}

func init() {
	indexCmd.Flags().BoolP("force", "f", false, "Force re-indexing of all files")
	rootCmd.AddCommand(indexCmd)
}
