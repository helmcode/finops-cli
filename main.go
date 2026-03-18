package main

import "github.com/helmcode/finops-cli/cmd"

// version is set by ldflags at build time.
var version = "dev"

func main() {
	cmd.SetVersion(version)
	cmd.Execute()
}
