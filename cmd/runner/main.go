package main

import (
	"fmt"
	"os"

	"github.com/smallnest/goclaw/cli"
)

var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

func main() {
	if err := cli.ConfigureApplication(cli.ApplicationRunner); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	cli.SetVersion(Version)
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
