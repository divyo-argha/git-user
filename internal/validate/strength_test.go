package validate

import "testing"

// TestPassphraseStrength covers the scoring behavior end to end — length
// alone (no character-class requirement), predictable-run discounting, and
// the blocklist cap. These expectations were verified against the actual
// implementation, not just designed on paper.
func TestPassphraseStrength(t *testing.T) {
	cases := []struct {
		name string
		pass string
		want StrengthLevel
	}{
		{"empty", "", VeryWeak},
		{"single char", "a", VeryWeak},
		{"short all-lower", "abcdef", VeryWeak},
		{"eight random lowercase", "kqzwrtjm", Fair},
		{"long random mixed classes", "K9$mQ2!vT8&nP5", Strong},
		{"repeated chars discounted", "aaaaaaaaaaaa", VeryWeak},
		{"sequential digits discounted", "12345678", VeryWeak},
		{"sequential letters discounted", "abcdefgh", VeryWeak},
		{"keyboard row discounted", "qwertyui", VeryWeak},
		{"blocklist entry, case-insensitive", "PASSWORD", Weak},
		{"blocklist entry with trailing digits normalized away", "password123", Weak},
		{"blocklist entry, plain", "letmein", Weak},
		{"long diceware-style phrase needs no symbols/uppercase", "correct horse battery staple", Strong},
		{"leetspeak variant of a blocklist entry", "p4ssw0rd", Weak},
		{"leetspeak + trailing year, a real-world common pattern", "wint3r2023", Weak},
		{"pluralized blocklist entry + trailing year", "Tigers2020", Weak},
		{"blocklist entry repeated twice", "adminadmin", Weak},
		{"composition-rule-satisfying but famously weak", "Password1!", Weak},
		{"common word + symbol + trailing year, looks complex but isn't", "Summer2024!", Weak},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := PassphraseStrength(c.pass)
			if got.Level != c.want {
				t.Errorf("PassphraseStrength(%q).Level = %v (%.1f bits), want %v", c.pass, got.Level, got.Bits, c.want)
			}
			if got.Message == "" {
				t.Errorf("PassphraseStrength(%q).Message is empty", c.pass)
			}
		})
	}
}

// TestIsCommonPassphraseAvoidsFalsePositives guards the normalization
// helpers added for leetspeak/pluralization/doubled-word detection against
// over-matching: a long, unrelated random string must never be flagged
// common just because some short substring or half of it happens to
// coincide with something in the blocklist.
func TestIsCommonPassphraseAvoidsFalsePositives(t *testing.T) {
	notCommon := []string{"xqzvbn7dkrpl", "K9$mQ2!vT8&nP5", "correct horse battery staple"}
	for _, p := range notCommon {
		if isCommonPassphrase(p) {
			t.Errorf("isCommonPassphrase(%q) = true, want false", p)
		}
	}
}

// TestPassphraseStrengthNeverRequiresCharacterClasses guards against a
// regression toward composition-rule-style scoring: an all-lowercase,
// all-digit, or all-symbol passphrase must still be able to reach every
// strength level on length alone.
func TestPassphraseStrengthNeverRequiresCharacterClasses(t *testing.T) {
	allLower := "correcthorsebatterystaplecorrecthorse" // long, single class, not a blocklist/pattern hit
	got := PassphraseStrength(allLower)
	if got.Level < Good {
		t.Errorf("expected a long single-character-class passphrase to still score well, got %v (%.1f bits)", got.Level, got.Bits)
	}
}

func TestPassphraseHintLineEmpty(t *testing.T) {
	if hint := PassphraseHintLine(""); hint != "" {
		t.Errorf("expected no hint for an empty passphrase, got %q", hint)
	}
}

func TestPassphraseHintLineNonEmpty(t *testing.T) {
	if hint := PassphraseHintLine("x"); hint == "" {
		t.Error("expected a hint for a non-empty passphrase, got empty string")
	}
}

// TestPassphraseAcceptsAnyLength is a regression guard for the SSH-key
// passphrase call sites, which now all pass minLen=0: Passphrase must never
// reject on length when minLen<=0, regardless of how short or long the
// input is — git-user places no restriction on SSH key passphrases.
func TestPassphraseAcceptsAnyLength(t *testing.T) {
	for _, pass := range []string{"", "a", "12", "short"} {
		if err := Passphrase(pass, 0); err != nil {
			t.Errorf("Passphrase(%q, 0) = %v, want nil (no length restriction)", pass, err)
		}
	}
}
