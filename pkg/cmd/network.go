package cmd

import (
	"github.com/spf13/cobra"
	"github.com/nikitastrike/gios/pkg/deploy"
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Stream live syslog for real-time debugging",
	Run: func(cmd *cobra.Command, args []string) {
		deploy.RunLogs()
	},
}

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Starts the background Go SSH service (Native mode)",
	Run: func(cmd *cobra.Command, args []string) {
		deploy.RunDaemon()
	},
}

var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Opens a persistent SSH tunnel to the device",
	Run: func(cmd *cobra.Command, args []string) {
		deploy.Connect()
	},
}

var disconnectCmd = &cobra.Command{
	Use:   "disconnect",
	Short: "Closes all active background connections",
	Run: func(cmd *cobra.Command, args []string) {
		deploy.Disconnect()
	},
}

var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Opens an interactive SSH terminal on the device",
	Run: func(cmd *cobra.Command, args []string) {
		deploy.RunShell()
	},
}

func init() {
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(daemonCmd)
	rootCmd.AddCommand(connectCmd)
	rootCmd.AddCommand(disconnectCmd)
	rootCmd.AddCommand(shellCmd)
}
