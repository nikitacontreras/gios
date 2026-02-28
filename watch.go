package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func runWatch() {
	fmt.Printf("\n%s[gios]%s Entering Pro Watch Mode (Auto-build on save)...\n", ColorCyan, ColorReset)
	fmt.Println("Gios will monitor your .go files and deploy automatically.")
	fmt.Println("--------------------------------------------------")

	cwd, _ := os.Getwd()
	lastMod := make(map[string]time.Time)

	// Initial Scan
	scan := func() bool {
		changed := false
		filepath.Walk(cwd, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !filepath.HasPrefix(path, cwd) {
				return nil
			}
			if filepath.Ext(path) != ".go" && filepath.Base(path) != "gios.json" {
				return nil
			}
			// Skip .gios folder
			if filepath.Base(filepath.Dir(path)) == ".gios" || filepath.Base(path)[0] == '.' {
				return nil
			}

			if info.ModTime().After(lastMod[path]) {
				if !lastMod[path].IsZero() {
					changed = true
					fmt.Printf("\n%s[watcher]%s Detected change in %s\n", ColorYellow, ColorReset, filepath.Base(path))
				}
				lastMod[path] = info.ModTime()
			}
			return nil
		})
		return changed
	}

	// Initial capture
	scan()
	
	// Initial Build/Run
	build()
	run()

	fmt.Println("\nWaiting for changes... (Ctrl+C to exit)")

	for {
		if scan() {
			// Clear screen for fresh output
			fmt.Print("\033[H\033[2J")
			fmt.Printf("\n%s[gios]%s Rebuilding project...\n", ColorCyan, ColorReset)
			build()
			run()
			fmt.Println("\nWatching for changes...")
		}
		time.Sleep(1 * time.Second)
	}
}
