package cmd

import (
	"github.com/spf13/cobra"
	"github.com/nikitastrike/gios/pkg/builder"
	"github.com/nikitastrike/gios/pkg/config"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Compile and sign the binary for target architecture",
	Run: func(cmd *cobra.Command, args []string) {
		builder.Build(config.UnsafeFlag)
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)
}
