package tui

import (
	"fmt"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/policyops"
)

func opPolicyWrite(repoRoot string, requireSigning bool, domains []string) (opResult, error) {
	if err := policyops.WritePolicy(repoRoot, requireSigning, domains); err != nil {
		return opResult{}, err
	}
	return opResult{detail: "Wrote .git-user-policy"}, nil
}

func opSignerAddIdentity(store *config.Store, repoRoot, name string) (opResult, error) {
	principals, keyBlob, err := policyops.ResolveSignerFromIdentity(store, name)
	if err != nil {
		return opResult{}, err
	}
	if err := policyops.UpsertSigner(repoRoot, principals, keyBlob); err != nil {
		return opResult{}, err
	}
	_, _ = policyops.WireAllowedSignersConfig(repoRoot)
	return opResult{detail: fmt.Sprintf("Added %s to .allowed-signers", strings.Join(principals, ", "))}, nil
}

func opSignerAddEmail(repoRoot, email, pubkeyFile string) (opResult, error) {
	principals, keyBlob, err := policyops.ResolveSignerFromEmail(email, pubkeyFile)
	if err != nil {
		return opResult{}, err
	}
	if err := policyops.UpsertSigner(repoRoot, principals, keyBlob); err != nil {
		return opResult{}, err
	}
	_, _ = policyops.WireAllowedSignersConfig(repoRoot)
	return opResult{detail: fmt.Sprintf("Added %s to .allowed-signers", strings.Join(principals, ", "))}, nil
}

func opSignerRemove(repoRoot, principal string) (opResult, error) {
	removed, err := policyops.RemoveSigner(repoRoot, principal)
	if err != nil {
		return opResult{}, err
	}
	if !removed {
		return opResult{detail: fmt.Sprintf("No entry found for %q", principal)}, nil
	}
	return opResult{detail: fmt.Sprintf("Removed %q from .allowed-signers", principal)}, nil
}
