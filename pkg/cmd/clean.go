package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/nikitastrike/gios/pkg/config"
	"github.com/nikitastrike/gios/pkg/utils"
)

var flushCache bool

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean build artifacts and optionally flush the transpiler cache",
	Run: func(cmd *cobra.Command, args []string) {
		cwd, _ := os.Getwd()
		
		fmt.Printf("%s[gios]%s Cleaning project workspace...\n", utils.ColorCyan, utils.ColorReset)
		
		if flushCache {
			cacheDir := filepath.Join(cwd, ".gios", "cache")
			os.RemoveAll(cacheDir)
			fmt.Printf(" - Transpiler cache flushed (%s)\n", cacheDir)
		}
		
		buildDir := filepath.Join(cwd, "build")
		os.RemoveAll(buildDir)
		fmt.Printf(" - Build folder removed (%s)\n", buildDir)
		
		conf, err := config.LoadConfigSafe()
		if err == nil && conf.Output != "" {
			os.Remove(conf.Output)
			fmt.Printf(" - Compiled binary removed (%s)\n", conf.Output)
		}
		
		fmt.Printf("%s[Success]%s Workspace clean.\n", utils.ColorGreen, utils.ColorReset)
	},
}

func init() {
	cleanCmd.Flags().BoolVar(&flushCache, "flush", false, "Flush the file transpilation cache entirely")
	rootCmd.AddCommand(cleanCmd)
}
