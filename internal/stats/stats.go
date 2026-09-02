package stats

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
)

type SortMode string

const (
	SortByCommits SortMode = "commits"
	SortByLines   SortMode = "lines"
)

type AuthorStat struct {
	DisplayName string
	Email       string
	Commits     int

	// --- Cryptographic signature status -----------------------------------
	// Derived EXCLUSIVELY from git's own `%G?` commit signature-status output
	// (see `git log --help`, PRETTY FORMATS). This is the only concept on
	// this struct backed by a real cryptographic check. It says nothing
	// about whether the author's email is a locally registered identity —
	// see VerifiedUser/IsRegisteredIdentity below for that.
	SignedCommits           int // %G? in {G, U, X, Y}: a valid signature from a currently-trusted key
	UnsignedCommits         int // %G? == N (or unrecognized): commit carries no signature at all
	RevokedSignatureCommits int // %G? == R: signature is valid but made by a key that has since been revoked — NOT trustworthy
	BadSignatureCommits     int // %G? == B: a signature is present but does not match (invalid/corrupt/tampered)
	UnverifiableCommits     int // %G? == E: git could not check the signature locally (no public key / no allowedSignersFile configured) — unknown status, not the same as "unsigned"

	CodeLinesAdded   int
	CodeLinesDeleted int
	NetCodeLines     int
	TotalLines       int

	// Line-level breakdown by the same crypto signature axis. "Signed" lines
	// belong to commits counted in SignedCommits; "Unsigned" lines belong to
	// commits counted in any of UnsignedCommits/RevokedSignatureCommits/
	// BadSignatureCommits/UnverifiableCommits (i.e. everything that is not a
	// currently-trusted good signature).
	SignedLinesAdded     int
	SignedLinesDeleted   int
	UnsignedLinesAdded   int
	UnsignedLinesDeleted int

	// --- Identity registration ---------------------------------------------
	// Derived EXCLUSIVELY from the local git-user config store
	// (internal/config). This is NOT a cryptographic check — it only says
	// whether this author's email matches an identity registered locally. A
	// commit can be identity-registered but never signed, and vice versa; do
	// not conflate this with the signature fields above.
	VerifiedUser   *config.User
	NameVariations []string
}

// IsRegisteredIdentity reports whether this author's email matches a locally
// registered identity in the git-user config store. This is an identity
// check, not a cryptographic one — it makes no claim about commit signing.
func (a AuthorStat) IsRegisteredIdentity() bool {
	return a.VerifiedUser != nil
}

// AuditRepository audits commit author identities in the git repository (defaults to sorting by commits).
func AuditRepository(store *config.Store, targetPath string) ([]AuthorStat, error) {
	return AuditRepositoryMode(store, targetPath, SortByCommits)
}

// prepareAllowedSignersFile writes a temporary SSH "allowed signers" file
// (see `git help gpg.ssh.allowedSignersFile`) so that git can cryptographically
// verify SSH-signed commits at all.
//
// Without ANY allowedSignersFile configured, git refuses to check SSH
// signatures and %G? reports "N" (no signature) even for a commit that
// genuinely carries a valid SSH signature — indistinguishable from a truly
// unsigned commit. Once a file is present, git verifies the signature
// against the embedded public key: commits signed by a key listed here (as a
// principal for a registered identity's email/alias) report "G" (trusted),
// while commits signed by any other key still report "U" (cryptographically
// valid, unknown signer) instead of the misleading "N". Both "G" and "U" are
// treated as SignedCommits — see AuthorStat doc comments.
//
// Returns the path to pass via `-c gpg.ssh.allowedSignersFile=<path>` and a
// cleanup function. Always returns a usable (possibly empty) file so SSH
// verification is unlocked even with no registered users.
func prepareAllowedSignersFile(store *config.Store) (path string, cleanup func(), err error) {
	f, err := os.CreateTemp("", "git-user-allowed-signers-*")
	if err != nil {
		return "", func() {}, err
	}
	cleanup = func() { os.Remove(f.Name()) }

	if store != nil {
		for _, u := range store.Users {
			if u.SSHKey == "" {
				continue
			}
			pubKey, readErr := os.ReadFile(u.SSHKey + ".pub")
			if readErr != nil {
				continue
			}
			principals := make([]string, 0, 1+len(u.Aliases))
			if u.Email != "" {
				principals = append(principals, u.Email)
			}
			principals = append(principals, u.Aliases...)
			if len(principals) == 0 {
				continue
			}
			fmt.Fprintf(f, "%s %s\n", strings.Join(principals, ","), strings.TrimSpace(string(pubKey)))
		}
	}

	if closeErr := f.Close(); closeErr != nil {
		cleanup()
		return "", func() {}, closeErr
	}
	return f.Name(), cleanup, nil
}

// AuditRepositoryMode audits commit author identities and calculates pure code line changes (excluding comments/blank lines).
func AuditRepositoryMode(store *config.Store, targetPath string, mode SortMode) ([]AuthorStat, error) {
	if !git.IsInRepo() {
		return nil, fmt.Errorf("not in a git repository")
	}

	allowedSignersPath, cleanupSigners, err := prepareAllowedSignersFile(store)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare SSH allowed_signers file: %w", err)
	}
	defer cleanupSigners()

	// %G? is git's own cryptographic signature-status flag for each commit —
	// the only real signal we rely on for signing status (see AuthorStat
	// doc comments for what each status letter means). gpg.ssh.allowedSignersFile
	// is set explicitly so SSH-signed commits are actually checked (see
	// prepareAllowedSignersFile) rather than silently reported as unsigned.
	args := []string{"-c", "gpg.ssh.allowedSignersFile=" + allowedSignersPath, "log", "--all", "--use-mailmap", "-p", "-U0", "--format=COMMIT|%an|%ae|%G?"}
	if targetPath != "" {
		args = append(args, "--", targetPath)
	}

	cmd := exec.Command("git", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve git log: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to retrieve git log: %w", err)
	}

	type emailGroup struct {
		email                   string
		nameCounts              map[string]int
		commits                 int
		signedCommits           int
		unsignedCommits         int
		revokedSignatureCommits int
		badSignatureCommits     int
		unverifiableCommits     int
		linesAdded              int
		linesDeleted            int
		signedLinesAdded        int
		signedLinesDeleted      int
		unsignedLinesAdded      int
		unsignedLinesDeleted    int
		matchedUser             *config.User
	}

	groups := make(map[string]*emailGroup)

	var currentGroup *emailGroup
	var isCurrentCommitSigned bool

	// Stream stdout line-by-line rather than buffering the whole `git log -p`
	// output up front — on a large repo that output can run into the
	// hundreds of MB, and every line is fully consumed on a single pass
	// anyway. The scanner buffer is raised well past its 64KB default since a
	// single diff line (e.g. a minified/bundled file) can exceed that.
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "COMMIT|") {
			header := strings.TrimPrefix(line, "COMMIT|")
			parts := strings.SplitN(header, "|", 3)
			name := ""
			email := ""
			sigStatus := ""
			if len(parts) > 0 {
				name = strings.TrimSpace(parts[0])
			}
			if len(parts) > 1 {
				email = strings.TrimSpace(parts[1])
			}
			if len(parts) > 2 {
				sigStatus = strings.TrimSpace(parts[2])
			}

			normEmail := strings.ToLower(email)
			if normEmail == "" {
				normEmail = "unknown"
			}

			var matched *config.User
			if store != nil {
				matched = store.FindUserByEmail(normEmail)
			}

			groupID := normEmail
			if matched != nil {
				groupID = "user:" + strings.ToLower(matched.Name)
			}

			grp, exists := groups[groupID]
			if !exists {
				primaryEmail := email
				if matched != nil && matched.Email != "" {
					primaryEmail = matched.Email
				}
				grp = &emailGroup{
					email:       primaryEmail,
					nameCounts:  make(map[string]int),
					matchedUser: matched,
				}
				groups[groupID] = grp
			}

			grp.commits++

			// Cryptographic signature classification — driven exclusively by
			// git's own %G? status for this commit. See AuthorStat doc
			// comments for the meaning of each bucket. Identity registration
			// (matched) is tracked entirely separately and never folded into
			// this classification.
			isCurrentCommitSigned = sigStatus == "G" || sigStatus == "U" || sigStatus == "X" || sigStatus == "Y"

			switch sigStatus {
			case "G", "U", "X", "Y":
				grp.signedCommits++
			case "R":
				grp.revokedSignatureCommits++
			case "B":
				grp.badSignatureCommits++
			case "E":
				grp.unverifiableCommits++
			default: // "N", or any empty/unrecognized value
				grp.unsignedCommits++
			}

			if name != "" {
				grp.nameCounts[name]++
			}
			currentGroup = grp
			continue
		}

		if currentGroup == nil {
			continue
		}

		// Skip diff header metadata
		if strings.HasPrefix(line, "diff --git") || strings.HasPrefix(line, "index ") ||
			strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ ") ||
			strings.HasPrefix(line, "@@ ") {
			continue
		}

		if strings.HasPrefix(line, "+") {
			codeText := line[1:]
			if !isCommentOrBlank(codeText) {
				currentGroup.linesAdded++
				if isCurrentCommitSigned {
					currentGroup.signedLinesAdded++
				} else {
					currentGroup.unsignedLinesAdded++
				}
			}
		} else if strings.HasPrefix(line, "-") {
			codeText := line[1:]
			if !isCommentOrBlank(codeText) {
				currentGroup.linesDeleted++
				if isCurrentCommitSigned {
					currentGroup.signedLinesDeleted++
				} else {
					currentGroup.unsignedLinesDeleted++
				}
			}
		}
	}
	scanErr := scanner.Err()

	if waitErr := cmd.Wait(); waitErr != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return nil, fmt.Errorf("failed to retrieve git log: %s", msg)
		}
		return nil, fmt.Errorf("failed to retrieve git log: %w", waitErr)
	}
	if scanErr != nil {
		return nil, fmt.Errorf("failed to read git log output: %w", scanErr)
	}

	var results []AuthorStat
	for _, grp := range groups {
		var names []string
		var topName string
		maxCount := -1

		for n, cnt := range grp.nameCounts {
			names = append(names, n)
			if cnt > maxCount {
				maxCount = cnt
				topName = n
			}
		}

		sort.Strings(names)

		displayName := topName
		if grp.matchedUser != nil {
			displayName = grp.matchedUser.Name
		}
		if displayName == "" {
			displayName = grp.email
		}

		netLines := grp.linesAdded - grp.linesDeleted
		totalLines := grp.linesAdded + grp.linesDeleted

		results = append(results, AuthorStat{
			DisplayName:             displayName,
			Email:                   grp.email,
			Commits:                 grp.commits,
			SignedCommits:           grp.signedCommits,
			UnsignedCommits:         grp.unsignedCommits,
			RevokedSignatureCommits: grp.revokedSignatureCommits,
			BadSignatureCommits:     grp.badSignatureCommits,
			UnverifiableCommits:     grp.unverifiableCommits,
			CodeLinesAdded:          grp.linesAdded,
			CodeLinesDeleted:        grp.linesDeleted,
			NetCodeLines:            netLines,
			SignedLinesAdded:        grp.signedLinesAdded,
			SignedLinesDeleted:      grp.signedLinesDeleted,
			UnsignedLinesAdded:      grp.unsignedLinesAdded,
			UnsignedLinesDeleted:    grp.unsignedLinesDeleted,
			TotalLines:              totalLines,
			VerifiedUser:            grp.matchedUser,
			NameVariations:          names,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		if mode == SortByLines {
			if results[i].NetCodeLines != results[j].NetCodeLines {
				return results[i].NetCodeLines > results[j].NetCodeLines
			}
		}
		return results[i].Commits > results[j].Commits
	})

	return results, nil
}

// isCommentOrBlank checks if a line of code is blank or a comment in common languages (Python, Go, JS, TS, C, Shell, HTML, SQL, etc.).
func isCommentOrBlank(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return true
	}

	// Single-line Python / Shell / YAML / Ruby comments
	if strings.HasPrefix(trimmed, "#") {
		return true
	}

	// Python multiline docstrings or strings used as comments
	if strings.HasPrefix(trimmed, `"""`) || strings.HasPrefix(trimmed, `'''`) {
		return true
	}

	// C-style / Go / Java / JS / TS / Rust / C# comments
	if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*/") || strings.HasPrefix(trimmed, "*") {
		return true
	}

	// HTML / XML comments
	if strings.HasPrefix(trimmed, "<!--") || strings.HasPrefix(trimmed, "-->") {
		return true
	}

	// SQL / Lua / Haskell comments
	if strings.HasPrefix(trimmed, "--") {
		return true
	}

	return false
}
