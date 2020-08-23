package main

import (
	"../../AtellaCli"
)

var (
	GitCommit string = "unknown"
	GoVersion string = "unknown"
	Version   string = "unknown"
	Service   string = "Atella-Cli"
	Arch      string = "unknown"
	Sys       string = "unknown"
	BinPrefix string = "/usr/bin"
)

func main() {
	AtellaCli.Service = Service
	AtellaCli.GitCommit = GitCommit
	AtellaCli.GoVersion = GoVersion
	AtellaCli.Sys = Sys
	AtellaCli.Arch = Arch
	AtellaCli.Version = Version
	AtellaCli.BinPrefix = BinPrefix
	AtellaCli.Command()
}
