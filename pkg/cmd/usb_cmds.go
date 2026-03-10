package cmd

import (
	"github.com/spf13/cobra"
	"github.com/nikitastrike/gios/pkg/diagnostic"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Display detailed hardware/battery info via USB",
	Run: func(cmd *cobra.Command, args []string) {
		diagnostic.RunInfo()
	},
}

var screenshotCmd = &cobra.Command{
	Use:   "screenshot",
	Short: "Capture a device screenshot (saved locally)",
	Run: func(cmd *cobra.Command, args []string) {
		diagnostic.RunScreenshot()
	},
}

var rebootCmd = &cobra.Command{
	Use:   "reboot",
	Short: "Force a reboot of the connected device",
	Run: func(cmd *cobra.Command, args []string) {
		diagnostic.RunReboot()
	},
}

var mountCmd = &cobra.Command{
	Use:   "mount",
	Short: "Mount required support files",
	Run: func(cmd *cobra.Command, args []string) {
		diagnostic.RunMount()
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(screenshotCmd)
	rootCmd.AddCommand(rebootCmd)
	rootCmd.AddCommand(mountCmd)
}
