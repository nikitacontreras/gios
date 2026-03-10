package cmd

import (
	"github.com/spf13/cobra"
	"github.com/nikitastrike/gios/pkg/transpiler"
)

var headersCmd = &cobra.Command{
	Use:   "headers [ProcessName]",
	Short: "Auto-extract ObjC headers from any process",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		transpiler.RunHeaders(args[0])
	},
}

var hookCmd = &cobra.Command{
	Use:   "hook [ClassName] [Method]",
	Short: "Generate DSL-based hooks for Go tweaks",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		transpiler.RunHook(args[0], args[1])
	},
}

var polyfillCmd = &cobra.Command{
	Use:   "polyfill",
	Short: "Compatibility patching for legacy systems",
	Run: func(cmd *cobra.Command, args []string) {
		transpiler.RunPolyfill()
	},
}

func init() {
	rootCmd.AddCommand(headersCmd)
	rootCmd.AddCommand(hookCmd)
	rootCmd.AddCommand(polyfillCmd)
}
