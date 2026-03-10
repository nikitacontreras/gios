package cmd

import (
	"github.com/spf13/cobra"
	"github.com/nikitastrike/gios/pkg/diagnostic"
)

var verboseFlag bool
var fixFlag bool

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Scan project for iOS compatibility (armv7/arm64)",
	Run: func(cmd *cobra.Command, args []string) {
		diagnostic.AnalyzeProject(verboseFlag)
	},
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose local environment (toolchain, SDks, USB)",
	Run: func(cmd *cobra.Command, args []string) {
		diagnostic.RunDoctor(fixFlag)
	},
}

func init() {
	analyzeCmd.Flags().BoolVarP(&verboseFlag, "verbose", "v", false, "Show detailed findings")
	doctorCmd.Flags().BoolVar(&fixFlag, "fix", false, "Attempt to fix missing tools automatically")
	rootCmd.AddCommand(analyzeCmd)
	rootCmd.AddCommand(doctorCmd)
}
