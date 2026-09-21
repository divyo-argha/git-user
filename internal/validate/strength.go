package validate

import (
	_ "embed"
	"fmt"
	"math"
	"regexp"
	"strings"
	"unicode"
)

// commonPassphrasesData is a small, self-authored list of well-known common
// passwords/passphrases (sequences, the word "password" and its leetspeak
// variants, default/product words, common names and hobby words) — the kind
// of thing that tops every published "most common passwords" study. It's
// compiled from general public knowledge, not copied from any third-party
// list, specifically to avoid any licensing ambiguity in an embedded file.
//
//go:embed data/common_passphrases.txt
var commonPassphrasesData string

var commonPassphrases = buildCommonPassphraseSet(commonPassphrasesData)

func buildCommonPassphraseSet(raw string) map[string]struct{} {
	set := make(map[string]struct{})
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		set[line] = struct{}{}
	}
	return set
}

// trailingSuffix strips a trailing run of digits/common punctuation so a
// blocklist check on "password123" or "letmein!" still matches "password"/
// "letmein" — the most common way people "customize" an otherwise-common
// passphrase.
var trailingSuffix = regexp.MustCompile(`[0-9!@#$.]+$`)

// leetSubstitutions reverses the handful of character substitutions used in
// nearly every real-world password-mangling ruleset (hashcat's best64.rule,
// John the Ripper's default rules, etc.) — "p4ssw0rd" and "wint3r2023"
// aren't meaningfully different from "password"/"winter" to an attacker
// running a dictionary attack with mangling rules, so they shouldn't be to
// this blocklist check either.
var leetSubstitutions = strings.NewReplacer(
	"4", "a", "@", "a",
	"3", "e",
	"1", "i", "!", "i",
	"0", "o",
	"5", "s", "$", "s",
	"7", "t",
)

func normalizeForBlocklist(pass string) string {
	s := strings.ToLower(strings.TrimSpace(pass))
	return trailingSuffix.ReplaceAllString(s, "")
}

// isCommonPassphrase checks pass against the blocklist under several cheap
// normalizations — plain, leetspeak-reversed, singularized, and "is this
// just a blocklist word repeated twice" — instead of only an exact
// (suffix-stripped) match. This is still nowhere near a full
// mangling-rule/dictionary-attack simulation (that's what real cracking
// tools are for), but it closes the most common gap where a structurally
// "complex-looking" passphrase is actually one of the most-guessed patterns
// in real password breaches.
func isCommonPassphrase(pass string) bool {
	base := normalizeForBlocklist(pass)
	leet := leetSubstitutions.Replace(base)

	candidates := []string{base, leet}
	for _, c := range []string{base, leet} {
		if len(c) > 3 && strings.HasSuffix(c, "s") {
			candidates = append(candidates, strings.TrimSuffix(c, "s"))
		}
		if n := len(c); n >= 4 && n%2 == 0 && c[:n/2] == c[n/2:] {
			candidates = append(candidates, c[:n/2])
		}
	}

	for _, c := range candidates {
		if _, ok := commonPassphrases[c]; ok {
			return true
		}
	}
	return false
}

// keyboardRuns lists common keyboard row sequences. A 3+ character run of a
// passphrase that also appears (forwards or backwards) in one of these is
// treated as low-entropy — a keyboard-adjacent pattern, not that many
// characters of real randomness.
var keyboardRuns = []string{
	"qwertyuiop", "asdfghjkl", "zxcvbnm",
	"1234567890", "!@#$%^&*()",
}

// assumedOfflineGuessesPerSec is an estimate of how many passphrase guesses
// per second a well-resourced offline attacker could try against a key
// encrypted with git-user's hardened KDF rounds (ssh.HardenedKDFRounds —
// 100), grounded in real published cracking-tool benchmarks rather than a
// guess:
//
//   - Hashcat's module for OpenSSH's bcrypt_pbkdf private-key format (added
//     in hashcat PR #4767, merged as mode 36800 — before that, hashcat could
//     only attack the legacy MD5-KDF PEM format, not the openssh-key-v1
//     format `ssh-keygen` has written by default since OpenSSH 7.8 (2018),
//     which is what git-user always generates) benchmarks at ~590 H/s at
//     the default 16 rounds on a modest single GPU (an RTX 4050 laptop GPU,
//     per benchmark discussion on that PR).
//   - bcrypt_pbkdf's cost scales ~linearly with rounds (confirmed by
//     hashcat's own cost-model documentation for this mode), so at
//     HardenedKDFRounds=100 that same modest GPU is already down to
//     ~590*16/100 ≈ 94 H/s.
//   - bcrypt-family KDFs are deliberately memory/sequential-bound and so
//     scale far more modestly across GPU tiers than fast hashes — e.g.
//     classic bcrypt (hashcat mode 3200) only reaches ~5-6 kH/s on a
//     flagship RTX 4090 at a comparable cost setting, vs. the same GPU's
//     ~300 GH/s on a fast hash like NTLM (a ~50,000,000x gap, not the
//     ~50x gap raw compute-core counts would suggest). Applying a
//     deliberately conservative ~20x "flagship GPU vs. modest laptop GPU"
//     multiplier for this hash family (rather than assuming anything close
//     to the raw hardware ratio) lands at ~1,900 H/s for a single strong
//     GPU — rounded to 2,000/sec here.
//
// This number exists only to make "time to crack" messaging concrete — it
// is not a security boundary, real attacker hardware and rented/distributed
// GPU clusters vary widely, and every message built from it is explicitly
// caveated as a rough estimate.
const assumedOfflineGuessesPerSec = 2000.0

// StrengthLevel classifies a passphrase's estimated resistance to offline
// guessing, from weakest to strongest.
type StrengthLevel int

const (
	VeryWeak StrengthLevel = iota
	Weak
	Fair
	Good
	Strong
)

func (l StrengthLevel) String() string {
	switch l {
	case VeryWeak:
		return "Very Weak"
	case Weak:
		return "Weak"
	case Fair:
		return "Fair"
	case Good:
		return "Good"
	case Strong:
		return "Strong"
	default:
		return "Unknown"
	}
}

// StrengthResult is a purely informational assessment of a passphrase.
// git-user enforces no length or composition requirement on SSH key
// passphrases — this never blocks anything, it only helps the person
// choosing one understand how much protection it would actually offer if
// the encrypted key file it protects were ever copied off the machine.
type StrengthResult struct {
	Level   StrengthLevel
	Label   string
	Bits    float64
	Message string
}

// PassphraseStrength estimates how resistant pass would be to an offline
// guessing attack against the key file it protects. An empty passphrase
// still gets a (Very Weak) result — callers that want to skip rating an
// intentionally-empty ("no passphrase") field should check for "" before
// calling, or use PassphraseHintLine, which does that for them.
func PassphraseStrength(pass string) StrengthResult {
	if pass == "" {
		return StrengthResult{Level: VeryWeak, Label: VeryWeak.String(), Message: levelMessage(VeryWeak, false)}
	}

	bits := estimateBits(pass)
	common := isCommonPassphrase(pass)

	level := levelForBits(bits)
	if common && level > Weak {
		level = Weak
	}

	message := levelMessage(level, common)
	if clause := crackTimeClause(level, bits); clause != "" {
		message += " " + clause
	}

	return StrengthResult{Level: level, Label: level.String(), Bits: bits, Message: message}
}

// PassphraseHintLine returns the ready-to-print strength message for pass,
// or "" for an empty passphrase — every optional-passphrase flow in
// git-user already treats empty as "skip", so there's nothing to rate.
func PassphraseHintLine(pass string) string {
	if pass == "" {
		return ""
	}
	return PassphraseStrength(pass).Message
}

func levelForBits(bits float64) StrengthLevel {
	switch {
	case bits < 28:
		return VeryWeak
	case bits < 36:
		return Weak
	case bits < 60:
		return Fair
	case bits < 80:
		return Good
	default:
		return Strong
	}
}

func levelMessage(level StrengthLevel, common bool) string {
	if common {
		return "This is a commonly used passphrase — it would likely be guessed early by anyone trying common ones first."
	}
	switch level {
	case VeryWeak:
		return "Very weak — this could be guessed in seconds if someone got hold of the key file. A longer or more unusual passphrase would help a lot."
	case Weak:
		return "Weak — a few more words or characters would meaningfully raise the bar."
	case Fair:
		return "Fair — reasonable for everyday use."
	case Good:
		return "Good — solid protection if this key file were ever copied."
	case Strong:
		return "Strong — excellent protection."
	default:
		return ""
	}
}

func crackTimeClause(level StrengthLevel, bits float64) string {
	if level < Fair {
		return ""
	}
	guesses := math.Pow(2, bits-1)
	seconds := guesses / assumedOfflineGuessesPerSec
	return fmt.Sprintf("(rough estimate: would take %s to guess offline, assuming a well-resourced attacker — actual attacker hardware varies)", humanizeDuration(seconds))
}

func humanizeDuration(seconds float64) string {
	switch {
	case seconds < 1:
		return "less than a second"
	case seconds < 60:
		return "seconds"
	case seconds < 3600:
		return "minutes"
	case seconds < 86400:
		return "hours"
	case seconds < 86400*30:
		return "days"
	case seconds < 86400*365:
		return "months"
	case seconds < 86400*365*100:
		return "years"
	default:
		return "centuries"
	}
}

// estimateBits computes a rough entropy estimate: an "effective length"
// (raw length, discounted for predictable runs — see discount*) multiplied
// by the number of bits contributed by each character given the character
// classes actually present. No class is ever required — an all-lowercase
// passphrase is scored purely on its (effective) length, since git-user
// places no composition requirement on SSH key passphrases.
func estimateBits(pass string) float64 {
	pool := poolSize(pass)
	bitsPerChar := math.Log2(float64(pool))
	effLen := effectiveLength(pass)
	return float64(effLen) * bitsPerChar
}

func poolSize(pass string) int {
	var hasLower, hasUpper, hasDigit, hasSymbol, hasOther bool
	for _, r := range pass {
		switch {
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsDigit(r):
			hasDigit = true
		case r < 128:
			hasSymbol = true
		default:
			hasOther = true
		}
	}
	size := 0
	if hasLower {
		size += 26
	}
	if hasUpper {
		size += 26
	}
	if hasDigit {
		size += 10
	}
	if hasSymbol {
		size += 33
	}
	if hasOther {
		size += 64
	}
	if size == 0 {
		size = 1
	}
	return size
}

// effectiveLength discounts runs of 3+ characters that are predictable
// (identical, ascending/descending by one codepoint, or a keyboard-row
// sequence) down to a weight of 2 for the whole run, instead of counting
// each character as a full unit of randomness. This is what makes "aaaaaaaa",
// "12345678", and "qwertyui" all score as Very Weak despite their raw
// length, without ever rejecting them outright.
func effectiveLength(pass string) int {
	runes := []rune(pass)
	n := len(runes)
	if n == 0 {
		return 0
	}

	weight := make([]float64, n)
	for i := range weight {
		weight[i] = 1
	}

	discountIdenticalAndSequentialRuns(runes, weight)
	discountKeyboardRuns(runes, weight)

	total := 0.0
	for _, w := range weight {
		total += w
	}
	eff := int(math.Round(total))
	if eff < 1 {
		eff = 1
	}
	return eff
}

func discountRun(weight []float64, start, end int) {
	runLen := end - start
	if runLen < 3 {
		return
	}
	target := 2.0 / float64(runLen)
	for i := start; i < end; i++ {
		if weight[i] > target {
			weight[i] = target
		}
	}
}

func discountIdenticalAndSequentialRuns(runes []rune, weight []float64) {
	n := len(runes)
	i := 0
	for i < n {
		j := i + 1
		mode := 0 // 0=undetermined, 1=identical, 2=ascending, 3=descending
		for j < n {
			delta := runes[j] - runes[j-1]
			ok := false
			switch {
			case delta == 0 && mode != 2 && mode != 3:
				mode = 1
				ok = true
			case delta == 1 && mode != 1 && mode != 3:
				mode = 2
				ok = true
			case delta == -1 && mode != 1 && mode != 2:
				mode = 3
				ok = true
			}
			if !ok {
				break
			}
			j++
		}
		if j-i >= 3 {
			discountRun(weight, i, j)
			i = j
		} else {
			i++
		}
	}
}

func discountKeyboardRuns(runes []rune, weight []float64) {
	n := len(runes)
	lowerRunes := make([]rune, n)
	for i, r := range runes {
		lowerRunes[i] = unicode.ToLower(r)
	}
	maxLen := 10
	if n < maxLen {
		maxLen = n
	}
	for length := maxLen; length >= 3; length-- {
		for start := 0; start+length <= n; start++ {
			window := string(lowerRunes[start : start+length])
			if !isASCII(window) {
				continue
			}
			for _, row := range keyboardRuns {
				if strings.Contains(row, window) || strings.Contains(reverseASCII(row), window) {
					discountRun(weight, start, start+length)
					break
				}
			}
		}
	}
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 128 {
			return false
		}
	}
	return true
}

func reverseASCII(s string) string {
	b := []byte(s)
	for l, h := 0, len(b)-1; l < h; l, h = l+1, h-1 {
		b[l], b[h] = b[h], b[l]
	}
	return string(b)
}
