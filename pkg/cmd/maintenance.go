package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/nikitastrike/gios/pkg/sdk"
	"github.com/nikitastrike/gios/pkg/utils"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Pull the latest version of GIOS from GitHub and rebuild",
		// Check if we are in a git repo
		if _, err := os.Stat(".git"); err == nil {
			fmt.Printf("%s[gios]%s Git repository detected. Pulling updates...\n", utils.ColorCyan, utils.ColorReset)
			
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
		} else {
			fmt.Printf("%s[gios]%s Binary installation detected. Fetching latest release...\n", utils.ColorCyan, utils.ColorReset)
			
			target := utils.GetTargetString()
			repo := "nikitacontreras/gios"
			downloadURL := fmt.Sprintf("https://github.com/%s/releases/latest/download/gios-%s", repo, target)
			if strings.Contains(target, "windows") {
				downloadURL += ".exe"
			}

			selfPath, err := os.Executable()
			if err != nil {
				fmt.Printf("%s[!] Error:%s Could not find self path: %v\n", utils.ColorRed, utils.ColorReset, err)
				return
			}

			// Rename current binary for safe replace
			oldPath := selfPath + ".old"
			if err := os.Rename(selfPath, oldPath); err != nil {
				fmt.Printf("%s[!] Error:%s Could not backup current binary: %v\n", utils.ColorRed, utils.ColorReset, err)
				return
			}

			if err := sdk.DownloadURLToFile(downloadURL, selfPath, true); err != nil {
				fmt.Printf("%s[!] Error:%s Update failed: %v\n", utils.ColorRed, utils.ColorReset, err)
				os.Rename(oldPath, selfPath) // Restore
				return
			}
			
			os.Chmod(selfPath, 0755)
			os.Remove(oldPath)
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
