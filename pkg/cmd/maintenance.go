package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/nikitastrike/gios/pkg/utils"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Pull the latest version of GIOS from GitHub and rebuild",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("%s[gios]%s Checking for updates...\n", utils.ColorCyan, utils.ColorReset)
		
		// 1. Pull
		pull := exec.Command("git", "pull")
		pull.Stdout = os.Stdout
		pull.Stderr = os.Stderr
		if err := pull.Run(); err != nil {
			fmt.Printf("%s[!] Error:%s Failed to pull from git: %v\n", utils.ColorRed, utils.ColorReset, err)
			return
		}

		// 2. Build
		fmt.Printf("%s[gios]%s Rebuilding GIOS binary...\n", utils.ColorCyan, utils.ColorReset)
		build := exec.Command("make", "build")
		build.Stdout = os.Stdout
		build.Stderr = os.Stderr
		if err := build.Run(); err != nil {
			fmt.Printf("%s[!] Error:%s Rebuild failed: %v\n", utils.ColorRed, utils.ColorReset, err)
			return
		}

		fmt.Printf("\n%s[Success]%s GIOS has been updated to the latest version!\n", utils.ColorGreen, utils.ColorReset)
	},
}

var diffCmd = &cobra.Command{
	Use:   "diff [file]",
	Short: "View transpilation changes for a specific file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		utils.RunDiff(args[0])
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(diffCmd)
}
