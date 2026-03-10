package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/nikitastrike/gios/pkg/builder"
	"github.com/nikitastrike/gios/pkg/config"
	"github.com/nikitastrike/gios/pkg/deploy"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Build and deploy the executable to the target device",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("[gios] Building & Deploying...")
		builder.Build(config.UnsafeFlag)
		deploy.Run(config.UnsafeFlag, false)
	},
}

var pkgCmd = &cobra.Command{
	Use:     "package",
	Aliases: []string{"deb"},
	Short:   "Build and create a Debian (.deb) package",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("[gios] Building before packaging...")
		builder.Build(config.UnsafeFlag)
		deploy.CreateDeb(config.UnsafeFlag)
	},
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Build, package, and install .deb to target",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("[gios] Starting installation pipeline...")
		builder.Build(config.UnsafeFlag)
		deploy.CreateDeb(config.UnsafeFlag)
		deploy.InstallDeb(config.UnsafeFlag)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(pkgCmd)
	rootCmd.AddCommand(installCmd)
}
