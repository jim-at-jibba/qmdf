package main

import (
	"runtime/debug"

	"github.com/jim-at-jibba/qmdf/cmd"
)

// version is set at build time via -ldflags "-X main.version=<tag>".
// When installed via `go install ...@v0.1.0`, Go embeds the module version
// in debug.BuildInfo, which we read as a fallback.
var version = "dev"

func main() {
	if version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
			version = info.Main.Version
		}
	}
	cmd.Execute(version)
}
