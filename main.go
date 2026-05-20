package main

import (
	"fmt"
	"os"

	"github.com/divyo-argha/git-user/cmd"
)

var (
	version = "dev"
	date    = "unknown"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v" || os.Args[1] == "version") {
		printVersion()
		os.Exit(0)
	}

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func printVersion() {
	// ANSI color codes
	cyan := "\033[36m"
	green := "\033[32m"
	reset := "\033[0m"
	
	fmt.Printf("%s✨ git-user%s %s%s%s %s• %s%s\n",
		green, reset,
		cyan, version, reset,
		green,
		date, reset)
}
