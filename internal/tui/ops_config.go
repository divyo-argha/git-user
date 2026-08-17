package tui

import (
	"fmt"
	"sort"

	"github.com/divyo-argha/git-user/internal/config"
)

// ── Custom git config ─────────────────────────────────────────────────────────

// opConfigList returns the custom git config keys for an identity.
func opConfigList(store *config.Store, name string) (opResult, error) {
	user := store.FindUser(name)
	if user == nil {
		return opResult{}, fmt.Errorf("identity %q not found", name)
	}
	if len(user.CustomConfig) == 0 {
		return opResult{detail: fmt.Sprintf("No custom config keys set for %q.", name), showReport: true}, nil
	}
	var keys []string
	for k := range user.CustomConfig {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	report := fmt.Sprintf("CUSTOM CONFIG FOR IDENTITY: %s\n\n", name)
	for _, k := range keys {
		report += fmt.Sprintf("  %s = %s\n", k, user.CustomConfig[k])
	}
	report += "\nThese are applied to git whenever this identity is active.\n"
	return opResult{detail: report, showReport: true}, nil
}

// opConfigSet sets a custom git config key for an identity.
func opConfigSet(store *config.Store, name, key, value string) (opResult, error) {
	if key == "" {
		return opResult{}, fmt.Errorf("config key is required")
	}
	user := store.FindUser(name)
	if user == nil {
		return opResult{}, fmt.Errorf("identity %q not found", name)
	}
	if user.CustomConfig == nil {
		user.CustomConfig = make(map[string]string)
	}
	user.CustomConfig[key] = value
	if err := config.Save(store); err != nil {
		return opResult{}, fmt.Errorf("saving config: %v", err)
	}
	if store.Current == name {
		_ = applyActiveCustomConfig(key, value, false)
	}
	return opResult{detail: fmt.Sprintf("Set config %q = %q for identity %q", key, value, name)}, nil
}

// opConfigUnset removes a custom git config key from an identity.
func opConfigUnset(store *config.Store, name, key string) (opResult, error) {
	if key == "" {
		return opResult{}, fmt.Errorf("config key is required")
	}
	user := store.FindUser(name)
	if user == nil {
		return opResult{}, fmt.Errorf("identity %q not found", name)
	}
	if user.CustomConfig != nil {
		delete(user.CustomConfig, key)
	}
	if err := config.Save(store); err != nil {
		return opResult{}, fmt.Errorf("saving config: %v", err)
	}
	if store.Current == name {
		_ = unsetActiveCustomConfig(key, false)
	}
	return opResult{detail: fmt.Sprintf("Unset config %q for identity %q", key, name)}, nil
}
