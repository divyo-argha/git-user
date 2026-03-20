package main

import (
	"os"

	"github.com/divyo-argha/git-user/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
