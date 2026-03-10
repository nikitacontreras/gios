package main

import (
	"github.com/nikitastrike/gios/pkg/cmd"
)

var (
	AppVersion = "v1.3.0"
	BuildTime  = "unknown"
)

func main() {
	cmd.Execute(AppVersion)
}
