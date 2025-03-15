package main

import (
	"fmt"
	"os"

	"github.com/warpbuilds/packer-plugin-lume/builder/lume"
	lume_export "github.com/warpbuilds/packer-plugin-lume/post-processor/lume-export"
	"github.com/warpbuilds/packer-plugin-lume/version"

	"github.com/hashicorp/packer-plugin-sdk/plugin"
)

func main() {
	pps := plugin.NewSet()
	pps.RegisterBuilder("cli", new(lume.Builder))
	pps.RegisterPostProcessor("export", new(lume_export.PostProcessor))
	pps.SetVersion(version.PluginVersion)
	err := pps.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
