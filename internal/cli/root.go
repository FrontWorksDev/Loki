package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "img-cli",
	Short:   "Loki - 画像圧縮CLIツール",
	Version: "0.1.0",
}

func init() {
	rootCmd.AddCommand(compressCmd)
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
