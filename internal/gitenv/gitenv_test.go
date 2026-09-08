package gitenv

import (
	"strings"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/keyring"
	zkeyring "github.com/zalando/go-keyring"
)

// TestVarsQuotesEmbeddedSingleQuote guards against a regression where an
// email local part containing an apostrophe (explicitly permitted by
// validate.Email, e.g. o'brien@example.com) corrupted GIT_CONFIG_PARAMETERS:
// naive `'%s'` wrapping left the quote unescaped, desyncing every parameter
// git parsed after it.
func TestVarsQuotesEmbeddedSingleQuote(t *testing.T) {
	u := &config.User{Name: "o'brien", Email: "o'brien@example.com"}
	params := Vars(u)["GIT_CONFIG_PARAMETERS"]

	if !strings.Contains(params, `user.name=o'\''brien`) {
		t.Errorf("expected escaped single quote in user.name, got: %s", params)
	}
	if !strings.Contains(params, `user.email=o'\''brien@example.com`) {
		t.Errorf("expected escaped single quote in user.email, got: %s", params)
	}

	// Every quote in the result must be part of a balanced '...' or the
	// escape sequence '\'' — an odd total count means the quoting desynced.
	if strings.Count(params, "'")%2 != 0 {
		t.Errorf("unbalanced quoting in GIT_CONFIG_PARAMETERS: %s", params)
	}
}

// TestVarsOmitsAskpassWithoutToken guards against wiring core.askpass in
// unconditionally — an identity with no stored HTTPS token must not get a
// GIT_CONFIG_PARAMETERS entry that would make every credential prompt (even
// for identities that only ever use SSH) shell out to git-user.
func TestVarsOmitsAskpassWithoutToken(t *testing.T) {
	u := &config.User{Name: "work", Email: "work@example.com"}
	params := Vars(u)["GIT_CONFIG_PARAMETERS"]
	if strings.Contains(params, "core.askpass") {
		t.Errorf("expected no core.askpass without a stored token, got: %s", params)
	}
}

// TestVarsWiresAskpassWithToken checks that an identity with a stored HTTPS
// token gets a core.askpass entry pointing back at this binary with its own
// name, so gitenv.AskpassCommand's output actually reaches
// GIT_CONFIG_PARAMETERS.
func TestVarsWiresAskpassWithToken(t *testing.T) {
	mockStore := map[string]string{"work": "ghp_abc123"}
	keyring.KeyringGet = func(service, user string) (string, error) {
		val, ok := mockStore[user]
		if !ok {
			return "", zkeyring.ErrNotFound
		}
		return val, nil
	}
	defer func() { keyring.KeyringGet = zkeyring.Get }()

	u := &config.User{Name: "work", Email: "work@example.com"}
	params := Vars(u)["GIT_CONFIG_PARAMETERS"]
	if !strings.Contains(params, "core.askpass=") {
		t.Fatalf("expected core.askpass in GIT_CONFIG_PARAMETERS, got: %s", params)
	}
	if !strings.Contains(params, "__askpass") {
		t.Errorf("expected the askpass command to invoke the __askpass helper, got: %s", params)
	}
	if !strings.Contains(params, "work") {
		t.Errorf("expected the identity name in the askpass command, got: %s", params)
	}
}
