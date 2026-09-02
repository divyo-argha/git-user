package gitenv

import (
	"strings"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
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
