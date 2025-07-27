package cmd

import (
	"fmt"

	"github.com/busybytelab.com/rag-cli/pkg/output"
	"github.com/spf13/cobra"
)

var (
	// Version is the version of the CLI tool
	Version = "dev"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display the version of the CLI tool",
	Long:  `Display the version of the RAG CLI tool.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("RAG CLI version %s\n", output.Highlight(Version))
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
