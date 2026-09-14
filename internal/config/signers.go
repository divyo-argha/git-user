package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const AllowedSignersFileName = ".allowed-signers"

type SignerEntry struct {
	Principals []string
	KeyBlob    string
}

func LoadAllowedSigners(repoRoot string) ([]SignerEntry, error) {
	f, err := os.Open(filepath.Join(repoRoot, AllowedSignersFileName))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var entries []SignerEntry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.SplitN(line, " ", 2)
		if len(fields) != 2 {
			continue
		}
		principals := strings.Split(fields[0], ",")
		entries = append(entries, SignerEntry{Principals: principals, KeyBlob: fields[1]})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}

func SaveAllowedSigners(repoRoot string, entries []SignerEntry) error {
	var b strings.Builder
	for _, e := range entries {
		fmt.Fprintf(&b, "%s %s\n", strings.Join(e.Principals, ","), e.KeyBlob)
	}
	return os.WriteFile(filepath.Join(repoRoot, AllowedSignersFileName), []byte(b.String()), 0644)
}

func hasPrincipal(e SignerEntry, principal string) bool {
	principal = strings.ToLower(principal)
	for _, p := range e.Principals {
		if strings.ToLower(p) == principal {
			return true
		}
	}
	return false
}

func UpsertSignerEntry(entries []SignerEntry, principals []string, keyBlob string) []SignerEntry {
	for i, e := range entries {
		for _, p := range principals {
			if hasPrincipal(e, p) {
				entries[i] = SignerEntry{Principals: principals, KeyBlob: keyBlob}
				return entries
			}
		}
	}
	return append(entries, SignerEntry{Principals: principals, KeyBlob: keyBlob})
}

func RemoveSignerEntries(entries []SignerEntry, principal string) []SignerEntry {
	var kept []SignerEntry
	for _, e := range entries {
		if !hasPrincipal(e, principal) {
			kept = append(kept, e)
		}
	}
	return kept
}
