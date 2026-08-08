package main

import (
	"bytes"
	"os"
	"testing"
)

func TestPrintVersion_Default(t *testing.T) {
	oldVersion := buildVersion
	buildVersion = "dev"
	t.Cleanup(func() { buildVersion = oldVersion })

	stdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = stdout })

	printVersion()
	w.Close()
	var outBuf bytes.Buffer
	outBuf.ReadFrom(r)

	output := outBuf.String()
	if output == "" {
		t.Error("printVersion() should output something")
	}
}

func TestPrintVersion_CustomBuild(t *testing.T) {
	oldVersion := buildVersion
	buildVersion = "v1.2.3"
	t.Cleanup(func() { buildVersion = oldVersion })

	stdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = stdout })

	printVersion()
	w.Close()
	var outBuf bytes.Buffer
	outBuf.ReadFrom(r)

	output := outBuf.String()
	if output == "" {
		t.Error("printVersion() should output something")
	}
}

func TestPrintVersion_EmptyBuild(t *testing.T) {
	oldVersion := buildVersion
	buildVersion = ""
	t.Cleanup(func() { buildVersion = oldVersion })

	stdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = stdout })

	printVersion()
	w.Close()
	var outBuf bytes.Buffer
	outBuf.ReadFrom(r)

	output := outBuf.String()
	if output == "" {
		t.Error("printVersion() should output something when buildVersion is empty")
	}
}

func TestCheckOrphanedKeys_SkipsNonInteractiveCommands(t *testing.T) {
	// Test that checkOrphanedKeys returns early for non-interactive commands
	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })

	skipCommands := []string{"--help", "-h", "help", "--version", "-v", "version", "completion", "prompt", "pubkey"}
	for _, cmd := range skipCommands {
		t.Run(cmd, func(t *testing.T) {
			os.Args = []string{"git-user", cmd}
			// Should not panic or error
			checkOrphanedKeys()
		})
	}
}
