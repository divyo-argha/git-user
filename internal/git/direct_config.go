package git

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GlobalConfigPath returns the canonical path to the user's global .gitconfig.
func GlobalConfigPath() string {
	if env := os.Getenv("GIT_CONFIG_GLOBAL"); env != "" {
		return env
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".gitconfig")
}

// parseFullKey breaks a dotted key like "user.name" or "includeif.gitdir/i:C:/Projects/.path"
// into its INI section header (e.g. "[user]" or "[includeif \"gitdir/i:C:/Projects/\"]") and leaf key name.
func parseFullKey(fullKey string) (sectionHeader string, key string, err error) {
	lastDot := strings.LastIndex(fullKey, ".")
	if lastDot == -1 {
		return "", "", fmt.Errorf("invalid git config key %q (must be section.key)", fullKey)
	}
	section := fullKey[:lastDot]
	key = fullKey[lastDot+1:]

	firstDot := strings.Index(section, ".")
	if firstDot == -1 {
		sectionHeader = "[" + strings.ToLower(section) + "]"
	} else {
		secName := strings.ToLower(section[:firstDot])
		subSec := section[firstDot+1:]
		sectionHeader = fmt.Sprintf("[%s %q]", secName, subSec)
	}
	return sectionHeader, strings.ToLower(key), nil
}

// getDirectConfig reads a value directly from an INI-style git config file without git.exe.
func getDirectConfig(configFile, fullKey string) (string, error) {
	targetSec, targetKey, err := parseFullKey(fullKey)
	if err != nil {
		return "", err
	}

	f, err := os.Open(configFile)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	currentSec := ""

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") || line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSec = line
			continue
		}
		if strings.EqualFold(currentSec, targetSec) {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				k := strings.ToLower(strings.TrimSpace(parts[0]))
				if k == targetKey {
					v := strings.TrimSpace(parts[1])
					if len(v) >= 2 && ((v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'')) {
						v = v[1 : len(v)-1]
					}
					return v, nil
				}
			}
		}
	}
	return "", fmt.Errorf("key %q not found in %s", fullKey, configFile)
}

// setDirectConfig writes or updates a key/value directly in an INI-style git config file.
func setDirectConfig(configFile, fullKey, value string) error {
	targetSec, targetKey, err := parseFullKey(fullKey)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(configFile), 0755); err != nil {
		return err
	}

	var lines []string
	if data, err := os.ReadFile(configFile); err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
	}

	currentSec := ""
	secIndex := -1
	keyIndex := -1

	for i, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSec = line
			if strings.EqualFold(currentSec, targetSec) {
				secIndex = i
			}
			continue
		}
		if strings.EqualFold(currentSec, targetSec) {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				k := strings.ToLower(strings.TrimSpace(parts[0]))
				if k == targetKey {
					keyIndex = i
					break
				}
			}
		}
	}

	newLine := fmt.Sprintf("\t%s = %s", targetKey, value)

	if keyIndex != -1 {
		lines[keyIndex] = newLine
	} else if secIndex != -1 {
		newLines := make([]string, 0, len(lines)+1)
		newLines = append(newLines, lines[:secIndex+1]...)
		newLines = append(newLines, newLine)
		newLines = append(newLines, lines[secIndex+1:]...)
		lines = newLines
	} else {
		if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) != "" {
			lines = append(lines, "")
		}
		lines = append(lines, targetSec, newLine)
	}

	out := strings.Join(lines, "\n") + "\n"
	return os.WriteFile(configFile, []byte(out), 0644)
}

// unsetDirectConfig removes a key from an INI-style git config file.
func unsetDirectConfig(configFile, fullKey string) error {
	targetSec, targetKey, err := parseFullKey(fullKey)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	var lines []string
	currentSec := ""

	for scanner.Scan() {
		rawLine := scanner.Text()
		line := strings.TrimSpace(rawLine)
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSec = line
			lines = append(lines, rawLine)
			continue
		}
		if strings.EqualFold(currentSec, targetSec) {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				k := strings.ToLower(strings.TrimSpace(parts[0]))
				if k == targetKey {
					continue
				}
			}
		}
		lines = append(lines, rawLine)
	}

	out := strings.Join(lines, "\n") + "\n"
	return os.WriteFile(configFile, []byte(out), 0644)
}
