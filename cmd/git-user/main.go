package main

import (
	"fmt"
	"os"

	"github.com/divyo-argha/git-user/internal/cli"
	"github.com/divyo-argha/git-user/internal/identity"
	"github.com/divyo-argha/git-user/internal/ui"
	"github.com/divyo-argha/git-user/internal/version"
)

var (
	buildVersion = "dev"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v", "version":
			printVersion()
			os.Exit(0)
		case "--update", "update":
			if err := cli.RunUpdate(); err != nil {
				os.Exit(1)
			}
			os.Exit(0)
		}
	}

	// Check for orphaned temporary keys on startup (skip for non-interactive commands)
	checkOrphanedKeys()

	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}

func printVersion() {
	v := buildVersion
	if v == "dev" || v == "" {
		v = version.Version
	}
	fmt.Printf("git-user %s\n", v)
}

func checkOrphanedKeys() {
	// Skip orphan check for non-interactive commands
	if len(os.Args) > 1 {
		cmdStr := os.Args[1]
		// Skip for these commands
		skipCommands := []string{"--help", "-h", "help", "--version", "-v", "version", "completion"}
		for _, skip := range skipCommands {
			if cmdStr == skip {
				return
			}
		}
	}

	// Try to detect orphaned keys
	manager, err := identity.NewManager()
	if err != nil {
		return // Silently fail - don't interrupt user
	}

	tempService := manager.GetTempService()
	orphanDetector := tempService.GetOrphanDetector()
	
	orphans, err := orphanDetector.Scan()
	if err != nil || len(orphans) == 0 {
		return // No orphans or error - continue silently
	}

	// Found orphaned keys - prompt user
	fmt.Println()
	ui.Warn(fmt.Sprintf("⚠️  Found %d abandoned temporary key(s):", len(orphans)))
	for _, orphan := range orphans {
		fmt.Printf("  • %s (created %s)\n", orphan.IdentityName, orphan.CreatedAt.Format("2006-01-02 15:04"))
	}
	fmt.Println()
	
	if ui.Confirm("Clean up these orphaned keys now?", true) {
		if err := orphanDetector.CleanupOrphans(orphans); err != nil {
			ui.Warn(fmt.Sprintf("Cleanup failed: %v", err))
		} else {
			ui.Success(fmt.Sprintf("✓ Cleaned up %d orphaned key(s)", len(orphans)))
		}
		fmt.Println()
	} else {
		ui.Info("You can clean them up later with: git-user security")
		fmt.Println()
	}
}
