package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/nikitastrike/gios/pkg/config"
	"github.com/nikitastrike/gios/pkg/utils"
)

var rootCmd = &cobra.Command{
	Use:   "gios",
	Short: "GIOS: The Modern Build System for Legacy and Modern iOS",
	Long: `GIOS (Go on iOS) is an ultra-fast, cross-platform CLI tool for building, 
packaging, and deploying Go-based tweaks and utilities to iOS devices.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute(version string) {
	rootCmd.Version = version
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&config.WatchFlag, "watch", "w", false, "Enable auto-redeploy on file save")
	rootCmd.PersistentFlags().BoolVar(&config.UnsafeFlag, "unsafe", false, "Force-transpile vendor dependencies")
	rootCmd.PersistentFlags().StringVar(&config.OutFlag, "out", "", "Override output binary filename")
	rootCmd.PersistentFlags().StringVar(&config.IPFlag, "ip", "", "Override device IP")
	rootCmd.PersistentFlags().BoolVar(&config.SyslogFlag, "syslog", false, "Use native USB syslog streaming")

	// Print custom header on all commands
	rootCmd.SetHelpTemplate(getHelpTemplate())
}

func getHelpTemplate() string {
	return utils.ColorCyan + utils.ColorBold + `
 dP""b8 88  dP"Yb  .dP"Y8 
dP   ` + "`" + `" 88 dP   Yb ` + "`" + `Ybo." 
Yb  "88 88 Yb   dP o.` + "`" + `Y8b 
 YboodP 88  YbodP  8bodP' 
` + utils.ColorReset + `
{{.Long}}

Usage:
  {{.CommandPath}}{{if .HasAvailableSubCommands}} [command]{{else}}{{if .HasAvailableLocalFlags}} [flags]{{end}}{{end}}

Available Commands:
{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}  {{rpad .Name .NamePadding}} {{.Short}}
{{end}}{{end}}
Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}

Use "{{.CommandPath}} [command] --help" for more information about a command.
`
}
