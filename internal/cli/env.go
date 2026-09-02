package cli

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/gitenv"
	"github.com/divyo-argha/git-user/internal/ui"
)

// ShellType represents the target shell dialect for environment scripts.
type ShellType string

const (
	ShellPosix      ShellType = "posix"      // bash, zsh, sh, ksh, dash
	ShellFish       ShellType = "fish"       // fish shell
	ShellPowerShell ShellType = "powershell" // pwsh, PowerShell
	ShellCmd        ShellType = "cmd"        // Windows Command Prompt
)

// detectShell determines the active or default shell syntax.
func detectShell(explicit string) ShellType {
	if explicit != "" {
		explicit = strings.ToLower(strings.TrimSpace(explicit))
		switch explicit {
		case "fish":
			return ShellFish
		case "pwsh", "powershell", "ps":
			return ShellPowerShell
		case "cmd", "batch":
			return ShellCmd
		case "bash", "zsh", "sh", "posix", "dash", "ksh":
			return ShellPosix
		}
	}

	shellEnv := strings.ToLower(os.Getenv("SHELL"))
	if strings.Contains(shellEnv, "fish") {
		return ShellFish
	}
	if strings.Contains(shellEnv, "pwsh") || strings.Contains(shellEnv, "powershell") {
		return ShellPowerShell
	}

	if runtime.GOOS == "windows" {
		if os.Getenv("PSModulePath") != "" {
			return ShellPowerShell
		}
		return ShellCmd
	}

	return ShellPosix
}

// EnvVars returns the map of Git environment variables for a given user identity.
func EnvVars(u *config.User) map[string]string {
	return gitenv.Vars(u)
}

func runEnv(args []string) error {
	unsetMode := false
	var targetName string
	var explicitShell string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "switch", "sw", "--switch", "--session", "-s":
			// Ignore subcommand and session flags passed from shell wrappers
			continue
		case "--unset", "-u":
			unsetMode = true
		case "--shell":
			if i+1 < len(args) {
				explicitShell = args[i+1]
				i++
			}
		case "--fish":
			explicitShell = "fish"
		case "--powershell", "--pwsh":
			explicitShell = "powershell"
		case "--cmd":
			explicitShell = "cmd"
		case "--bash", "--zsh", "--posix", "--sh":
			explicitShell = "posix"
		case "-h", "--help":
			runSubcommandHelp("env")
			return nil
		default:
			if !strings.HasPrefix(args[i], "-") && targetName == "" {
				targetName = args[i]
			}
		}
	}

	st := detectShell(explicitShell)

	if unsetMode {
		fmt.Fprintf(os.Stderr, "✔ Cleared terminal session override. Returned to global profile.\n")
		return printUnsetScript(st)
	}

	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	if targetName == "" {
		targetName = store.Current
	}

	if targetName == "" {
		ui.Errorf("no identity specified and no active profile found")
		return fmt.Errorf("missing identity")
	}

	user := store.FindUser(targetName)
	if user == nil {
		ui.Errorf("identity %q not found", targetName)
		return fmt.Errorf("identity not found: %s", targetName)
	}

	fmt.Fprintf(os.Stderr, "✔ Switched terminal session identity to %q (%s)\n", user.Name, user.Email)
	vars := EnvVars(user)
	return printExportScript(st, vars)
}

func printExportScript(st ShellType, vars map[string]string) error {
	keys := []string{
		"GIT_AUTHOR_NAME",
		"GIT_AUTHOR_EMAIL",
		"GIT_COMMITTER_NAME",
		"GIT_COMMITTER_EMAIL",
		"GIT_USER_SESSION",
		"GIT_SSH_COMMAND",
		"GIT_CONFIG_PARAMETERS",
	}

	for _, k := range keys {
		val, exists := vars[k]
		if !exists || val == "" {
			continue
		}

		switch st {
		case ShellFish:
			fmt.Printf("set -gx %s %s;\n", k, fishQuote(val))
		case ShellPowerShell:
			fmt.Printf("$env:%s = %s\n", k, powershellQuote(val))
		case ShellCmd:
			fmt.Printf("set \"%s=%s\"\n", k, cmdSanitize(val))
		default: // POSIX (bash/zsh/sh)
			fmt.Printf("export %s=%s\n", k, posixQuote(val))
		}
	}
	return nil
}

// posixQuote and fishQuote wrap a value in single quotes for their
// respective shells so the caller's `eval "$(git-user env ...)"` can't run
// command substitution or expansion on a value git-user doesn't control —
// e.g. an identity name/email pulled in via bundle import or sync. Go's %q
// verb was used here previously, but it escapes for Go string-literal
// syntax, not shell syntax: it leaves $, `, and (in fish) parentheses
// unescaped, so a crafted value — validate.Email's local-part regex allows
// `, $, and / — would execute on eval instead of just being exported.
func posixQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// fishQuote escapes for fish's single-quote rules, where a backslash is only
// special immediately before another backslash or a single quote.
func fishQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `\'`)
	return "'" + s + "'"
}

// powershellQuote escapes for PowerShell single-quoted strings, which unlike
// double-quoted ones perform no interpolation at all — the only special
// sequence is a doubled single quote for a literal one.
func powershellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// cmdSanitize strips characters cmd.exe's `set "VAR=value"` form can't
// safely carry: an embedded double quote would close the quoted region
// early, re-exposing the rest of the value to & | < > parsing, and a CR/LF
// would split it into additional batch commands. Neither is meaningful data
// to preserve in a git identity name/email/path.
func cmdSanitize(s string) string {
	return strings.NewReplacer(`"`, "", "\r", "", "\n", "").Replace(s)
}

func printUnsetScript(st ShellType) error {
	keys := []string{
		"GIT_AUTHOR_NAME",
		"GIT_AUTHOR_EMAIL",
		"GIT_COMMITTER_NAME",
		"GIT_COMMITTER_EMAIL",
		"GIT_USER_SESSION",
		"GIT_SSH_COMMAND",
		"GIT_CONFIG_PARAMETERS",
	}

	switch st {
	case ShellFish:
		for _, k := range keys {
			fmt.Printf("set -e %s;\n", k)
		}
	case ShellPowerShell:
		for _, k := range keys {
			fmt.Printf("Remove-Item env:%s -ErrorAction SilentlyContinue\n", k)
		}
	case ShellCmd:
		for _, k := range keys {
			fmt.Printf("set %s=\n", k)
		}
	default: // POSIX
		fmt.Printf("unset %s\n", strings.Join(keys, " "))
	}
	return nil
}
