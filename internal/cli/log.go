package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
)

// runLog prints the identity-switch audit log recorded by AppendSwitchLog.
func runLog(args []string) error {
	limit := 20
	plain := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-n", "--limit":
			if i+1 < len(args) {
				if n, err := strconv.Atoi(args[i+1]); err == nil && n > 0 {
					limit = n
				}
				i++
			}
		case "--all":
			limit = 0
		case "--plain":
			plain = true
		}
	}

	entries, err := config.ReadSwitchLog()
	if err != nil {
		ui.Errorf("reading switch log: %v", err)
		return err
	}

	if len(entries) == 0 {
		ui.Info("No identity switches recorded yet.")
		return nil
	}

	if limit > 0 && len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}

	if !plain {
		ui.Banner("IDENTITY SWITCH LOG")
		fmt.Println()
	}

	for _, line := range entries {
		parts := strings.SplitN(line, "\t", 3)
		var ts, name, repo string
		if len(parts) > 0 {
			ts = parts[0]
		}
		if len(parts) > 1 {
			name = parts[1]
		}
		if len(parts) > 2 {
			repo = parts[2]
		}
		if plain {
			fmt.Printf("%s\t%s\t%s\n", ts, name, repo)
			continue
		}
		fmt.Printf("  %s  %-20s  %s\n", ts, name, repo)
	}

	return nil
}
