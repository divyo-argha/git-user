package cli

import (
	"fmt"
	"strings"

	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runFixRemote(args []string) error {
	implicit := false
	explicit := false
	global := false

	for _, a := range args {
		switch a {
		case "--implicit", "-i", "--push-only", "-p":
			implicit = true
		case "--explicit", "--rewrite":
			explicit = true
		case "--global", "-g":
			global = true
		case "-h", "--help":
			printFixRemoteHelp()
			return nil
		default:
			ui.Errorf("unknown flag: %s", a)
			ui.Info("usage: git-user fix-remote [--implicit] [--global] [--explicit]")
			return fmt.Errorf("unknown flag: %s", a)
		}
	}

	if implicit && explicit {
		ui.Error("cannot specify both --implicit and --explicit")
		return fmt.Errorf("conflicting flags")
	}

	if explicit {
		return runFixRemoteExplicit()
	}

	if implicit || global {
		return runFixRemoteImplicit(global)
	}

	// Neither flag specified:
	if !ui.IsTTY() {
		// Non-interactive (tests, scripts, CI): preserve backwards compatibility with explicit rewrite
		return runFixRemoteExplicit()
	}

	// Interactive terminal: offer choice
	choices := []string{
		"Implicit SSH push for this repo (keep HTTPS fetch, transparent SSH push) [Recommended]",
		"Implicit SSH push globally (applies to github.com, gitlab.com, bitbucket.org)",
		"Explicit rewrite (rewrite remote URLs to git@<host>: in git config)",
	}
	sel, err := ui.Select("How would you like to route Git remotes over SSH?", choices)
	if err != nil {
		return err
	}

	switch sel {
	case 0:
		return runFixRemoteImplicit(false)
	case 1:
		return runFixRemoteImplicit(true)
	case 2:
		return runFixRemoteExplicit()
	default:
		return nil
	}
}

func printFixRemoteHelp() {
	fmt.Println("Usage: git-user fix-remote [flags]")
	fmt.Println()
	fmt.Println("Routes repository pushes over SSH instead of HTTPS.")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -i, --implicit, -p, --push-only   Route push over SSH via pushInsteadOf (keeps HTTPS fetch URL untouched)")
	fmt.Println("  -g, --global                      Apply pushInsteadOf globally for GitHub, GitLab, Bitbucket")
	fmt.Println("      --explicit, --rewrite         Rewrite remote URLs in git config from https:// to git@...")
	fmt.Println("  -h, --help                        Show this help")
}

func runFixRemoteImplicit(global bool) error {
	if global {
		hosts := git.DefaultPushInsteadOfHosts()
		for _, host := range hosts {
			if err := git.ConfigurePushInsteadOf(host, false); err != nil {
				ui.Errorf("failed to configure global pushInsteadOf for %s: %v", host, err)
				return err
			}
		}
		if git.IsInRepo() {
			if remotes, err := git.ListRemotes(); err == nil {
				for _, r := range remotes {
					if u, err := git.GetRemoteURL(r); err == nil {
						if h := extractHostFromURL(u); h != "" && !containsStr(hosts, h) {
							_ = git.ConfigurePushInsteadOf(h, false)
							hosts = append(hosts, h)
						}
					}
				}
			}
		}
		ui.Success(fmt.Sprintf("Configured global implicit SSH push for: %s", strings.Join(hosts, ", ")))
		ui.Info("Pushes to HTTPS URLs will transparently route over SSH while keeping remote URLs untouched.")
		ui.Info("Try: git push")
		return nil
	}

	if !git.IsInRepo() {
		ui.Error("not in a git repository")
		return fmt.Errorf("not in a git repository")
	}

	remotes, err := git.ListRemotes()
	if err != nil || len(remotes) == 0 {
		ui.Error("no remotes found in current repository")
		return fmt.Errorf("no remotes found")
	}

	var configuredHosts []string
	configuredRemotes := 0

	for _, remote := range remotes {
		url, err := git.GetRemoteURL(remote)
		if err != nil || !strings.HasPrefix(url, "https://") {
			continue
		}
		host := extractHostFromURL(url)
		if host == "" {
			continue
		}
		if err := git.ConfigurePushInsteadOf(host, true); err != nil {
			ui.Warn(fmt.Sprintf("%s: failed to configure pushInsteadOf for %s: %v", remote, host, err))
			continue
		}
		if !containsStr(configuredHosts, host) {
			configuredHosts = append(configuredHosts, host)
		}
		configuredRemotes++
		pushURL, _ := git.GetPushRemoteURL(remote)
		ui.Success(fmt.Sprintf("%s: %s (push: %s)", remote, url, pushURL))
	}

	if configuredRemotes == 0 {
		ui.Info("All remotes already use SSH (or no HTTPS remotes found)")
	} else {
		fmt.Println()
		ui.Success(fmt.Sprintf("Configured implicit SSH push for %d remote(s)", configuredRemotes))
		ui.Info("Remote URLs remain untouched in git remote -v, while git push transparently uses SSH.")
		ui.Info("Try: git push")
	}

	return nil
}

func runFixRemoteExplicit() error {
	results, err := git.ConvertRemotesToSSH()
	if err != nil {
		ui.Error(err.Error())
		return err
	}

	converted := 0
	for _, r := range results {
		switch {
		case r.ConvertFailed:
			ui.Warn(fmt.Sprintf("%s: could not convert %s", r.Remote, r.OldURL))
		case r.UpdateFailed:
			ui.Warn(fmt.Sprintf("%s: failed to update", r.Remote))
		case r.Converted:
			ui.Success(fmt.Sprintf("%s: %s → %s", r.Remote, r.OldURL, r.NewURL))
			converted++
		}
	}

	if converted == 0 {
		ui.Info("All remotes already use SSH")
	} else {
		fmt.Println()
		ui.Success(fmt.Sprintf("Converted %d remote(s) to SSH", converted))
		ui.Info("Try: git push")
	}

	return nil
}

func extractHostFromURL(rawURL string) string {
	if !strings.HasPrefix(rawURL, "https://") && !strings.HasPrefix(rawURL, "http://") {
		return ""
	}
	rawURL = strings.TrimPrefix(rawURL, "https://")
	rawURL = strings.TrimPrefix(rawURL, "http://")
	parts := strings.Split(rawURL, "/")
	if len(parts) == 0 || parts[0] == "" {
		return ""
	}
	host := parts[0]
	if idx := strings.Index(host, "@"); idx != -1 {
		host = host[idx+1:]
	}
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}
	return host
}

func containsStr(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}
