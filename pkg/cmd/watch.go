package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/nikitastrike/gios/pkg/builder"
	"github.com/nikitastrike/gios/pkg/config"
	"github.com/nikitastrike/gios/pkg/deploy"
	"github.com/nikitastrike/gios/pkg/utils"
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Monitor and redeploy on every file save",
	Run: func(cmd *cobra.Command, args []string) {
		RunWatch()
	},
}

func RunWatch() {
	fmt.Printf("\n%s[gios]%s Entering Pro Watch Mode (Auto-build on save)...\n", utils.ColorCyan, utils.ColorReset)
	fmt.Println("Gios will monitor your .go files and deploy automatically.")
	fmt.Println("--------------------------------------------------")

	cwd, _ := os.Getwd()
	lastMod := make(map[string]time.Time)

	scan := func() bool {
		changed := false
		filepath.Walk(cwd, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			if filepath.Ext(path) != ".go" && filepath.Base(path) != "gios.json" {
				return nil
			}
			if strings.Contains(path, ".gios") || strings.Contains(path, "vendor") {
				return nil
			}

			if info.ModTime().After(lastMod[path]) {
				if !lastMod[path].IsZero() {
					changed = true
					fmt.Printf("\n%s[watcher]%s Detected change in %s\n", utils.ColorYellow, utils.ColorReset, filepath.Base(path))
				}
				lastMod[path] = info.ModTime()
			}
			return nil
		})
		return changed
	}

	scan()
	builder.Build(config.UnsafeFlag)
	deploy.Run(config.UnsafeFlag, true)

	fmt.Println("\nWaiting for changes... (Ctrl+C to exit)")

	for {
		if scan() {
			fmt.Print("\033[H\033[2J")
			fmt.Printf("\n%s[gios]%s Rebuilding project...\n", utils.ColorCyan, utils.ColorReset)
			builder.Build(config.UnsafeFlag)
			deploy.Run(config.UnsafeFlag, true)
			fmt.Println("\nWatching for changes...")
		}
		time.Sleep(1 * time.Second)
	}
}

func init() {
	rootCmd.AddCommand(watchCmd)
}
