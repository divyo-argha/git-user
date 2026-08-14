package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// switchLogMaxEntries caps the audit log so it can't grow unbounded over the
// life of a long-running machine.
const switchLogMaxEntries = 500

// SwitchLogPath returns the path to the identity-switch audit log, stored
// alongside the main config file.
func SwitchLogPath() string {
	return filepath.Join(filepath.Dir(ConfigPath()), "switch.log")
}

// AppendSwitchLog records a successful identity switch for later review via
// `git-user log`. It is best-effort and safe to ignore: a logging failure
// must never block the switch itself.
func AppendSwitchLog(identity, repoPath string) error {
	path := SwitchLogPath()
	lines, err := readSwitchLog(path)
	if err != nil {
		return err
	}

	ts := time.Now().UTC().Format(time.RFC3339)
	if repoPath == "" {
		repoPath = "-"
	}
	lines = append(lines, fmt.Sprintf("%s\t%s\t%s", ts, identity, repoPath))

	if len(lines) > switchLogMaxEntries {
		lines = lines[len(lines)-switchLogMaxEntries:]
	}

	content := strings.Join(lines, "\n") + "\n"
	return os.WriteFile(path, []byte(content), 0600)
}

// ReadSwitchLog returns the recorded switch-log entries, oldest first. Each
// entry is a tab-separated "<RFC3339 timestamp>\t<identity>\t<repo path>"
// line.
func ReadSwitchLog() ([]string, error) {
	return readSwitchLog(SwitchLogPath())
}

func readSwitchLog(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	text := strings.TrimRight(string(data), "\n")
	if text == "" {
		return nil, nil
	}
	return strings.Split(text, "\n"), nil
}
