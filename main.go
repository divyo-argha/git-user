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
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v", "version":
			printVersion()
			os.Exit(0)
		case "--update", "update":
			if err := cmd.RunUpdate(); err != nil {
				os.Exit(1)
			}
			os.Exit(0)
		}
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
