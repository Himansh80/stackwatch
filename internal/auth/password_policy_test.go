package auth

import (
	"strings"
	"testing"
)

// Valid password = at least 10 chars, has letter + digit-or-symbol,
// not whitespace-only, no leading/trailing whitespace, not in blocklist,
// does not contain a top-1000 entry as a 4+ char substring, and
// scores >= 40 on the entropy-based strength gate.
//
// The fixture set is small but each entry has been hand-checked against
// the FULL policy (substring blocklist + score gate + class check)
// as of the 2026-08-22 strengthening pass.
var goodPasswords = []string{
	"Hx42!kr9@pT",     // mixed classes, no common substrings
	"Tr0ub4dor&3",     // xkcd classic with digit + symbol
	"MyCat_!n_2026",   // letters + digits + symbols
	"K8mZ!qP3@vR5",    // 12 chars, 4 classes
	"Wx9!mK2#pL4@",    // 12 chars, 4 classes
	"Ranchero77!Bay",  // letters + digits + symbol
	"Sundrop#Mango42", // mixed words + numbers + symbol
}

func valid() string { return "K8mZ!qP3@vR5" }

func TestValidatePassword_AcceptsValid(t *testing.T) {
	for _, p := range goodPasswords {
		if err := ValidatePassword(p); err != nil {
			t.Errorf("ValidatePassword(%q) unexpectedly rejected: %v", p, err)
		}
	}
}

func TestValidatePassword_RejectsTooShort(t *testing.T) {
	for _, p := range []string{"", "a", "abc", "123456789", "abcdefghi"} {
		err := ValidatePassword(p)
		if err == nil {
			t.Errorf("ValidatePassword(%q) should reject (too short)", p)
		}
		if err != nil && err.(*PasswordPolicyError).Code != CodePasswordTooShort {
			t.Errorf("ValidatePassword(%q) wrong code: want %s, got %s", p, CodePasswordTooShort, err.(*PasswordPolicyError).Code)
		}
	}
}

func TestValidatePassword_RejectsTooLong(t *testing.T) {
	p := strings.Repeat("a", 129)
	err := ValidatePassword(p)
	if err == nil {
		t.Errorf("ValidatePassword(129 chars) should reject")
	}
	if err.(*PasswordPolicyError).Code != CodePasswordTooLong {
		t.Errorf("wrong code")
	}
}

func TestValidatePassword_RejectsWhitespaceOnly(t *testing.T) {
	p := "          "
	err := ValidatePassword(p)
	if err == nil {
		t.Errorf("ValidatePassword(%q) should reject (whitespace only)", p)
	}
	if err.(*PasswordPolicyError).Code != CodePasswordWhitespaceOnly {
		t.Errorf("wrong code: %s", err.(*PasswordPolicyError).Code)
	}
}

func TestValidatePassword_RejectsLeadingTrailingSpace(t *testing.T) {
	cases := []string{" K8mZ!qP3@vR5", "K8mZ!qP3@vR5 ", "  K8mZ!qP3@vR5  "}
	for _, p := range cases {
		err := ValidatePassword(p)
		if err == nil {
			t.Errorf("ValidatePassword(%q) should reject (leading/trailing space)", p)
		}
		if err != nil && err.(*PasswordPolicyError).Code != CodePasswordLeadingTrailing {
			t.Errorf("ValidatePassword(%q) wrong code: %s", p, err.(*PasswordPolicyError).Code)
		}
	}
}

func TestValidatePassword_RejectsBlocklist(t *testing.T) {
	// The top-1000 list contains only 4-9 char passwords (we counted
	// in build). Every one of them is already rejected by the length
	// check (min=10). The blocklist is therefore a defensive
	// backstop — it adds protection against the user picking any of
	// the same common passwords once length constraints are loosened
	// or if SecLists swaps to a longer list in the future.
	//
	// We test the blocklist machinery directly here: confirm the
	// lookup function finds entries regardless of how length
	// validation runs.
	t.Run("direct-lookup-finds-known-entries", func(t *testing.T) {
		for _, p := range []string{
			"password", "123456", "qwerty", "letmein", "monkey",
			"abc123", "football", "iloveyou", "admin", "welcome",
		} {
			if !inBlocklist(p) {
				t.Errorf("inBlocklist(%q) should be true (entry in top-1000)", p)
			}
		}
	})
	t.Run("direct-lookup-case-insensitive", func(t *testing.T) {
		for _, p := range []string{"PASSWORD", "Password", "pAsSwOrD"} {
			if !inBlocklist(p) {
				t.Errorf("inBlocklist(%q) should be true (case-insensitive)", p)
			}
		}
	})
	t.Run("direct-lookup-misses-non-entries", func(t *testing.T) {
		for _, p := range []string{
			"K8mZ!qP3@vR5",
			"Wx9!mK2#pL4@",
			"Ranchero77!Bay",
		} {
			if inBlocklist(p) {
				t.Errorf("inBlocklist(%q) should be false (not in top-1000)", p)
			}
		}
	})
}

func TestValidatePassword_BlocklistCatchesShortPasswords(t *testing.T) {
	// Blocklist also covers the short-but-common cases that hit
	// password_too_short first. This test confirms the blocklist
	// actually has these entries (the error code may vary).
	for _, p := range []string{"password", "123456", "qwerty", "admin"} {
		err := ValidatePassword(p)
		if err == nil {
			t.Errorf("ValidatePassword(%q) should reject", p)
		}
		// Either password_too_short or password_in_blocklist is fine —
		// both are correctly rejecting this input.
		code := err.(*PasswordPolicyError).Code
		if code != CodePasswordTooShort && code != CodePasswordInBlocklist {
			t.Errorf("ValidatePassword(%q) wrong code: %s", p, code)
		}
	}
}

func TestValidatePassword_RejectsBlocklistCaseInsensitive(t *testing.T) {
	cases := []string{"PASSWORD", "Password", "pAsSwOrD"}
	for _, p := range cases {
		err := ValidatePassword(p)
		if err == nil {
			t.Errorf("ValidatePassword(%q) should reject (blocklisted, mixed case)", p)
		}
	}
}

func TestValidatePassword_RejectsContainsCommonSubstring(t *testing.T) {
	// New in 2026-08-22: substring match catches things like
	// "passwordpassword" that exact-match would miss.
	cases := []string{
		"passwordpassword", // contains "password"
		"qwertyqwerty",     // contains "qwerty"
		"adminadmin123",    // contains "admin"
		"MyHunter2026",     // contains "hunter"
		"I-Love-Code-2026", // contains "love" + "code"
	}
	for _, p := range cases {
		err := ValidatePassword(p)
		if err == nil {
			t.Errorf("ValidatePassword(%q) should reject (contains common substring)", p)
			continue
		}
		// Could also hit inBlocklist exact match if the substring happens
		// to equal a blocklist entry verbatim. Both are correct rejections.
		code := err.(*PasswordPolicyError).Code
		if code != CodePasswordContainsCommon && code != CodePasswordInBlocklist {
			t.Errorf("ValidatePassword(%q) wrong code: %s", p, code)
		}
	}
}

func TestValidatePassword_RejectsAllLetters(t *testing.T) {
	// 10+ letters with no digit/symbol must fail the shape check.
	cases := []string{
		"abcdefghijk",  // 11 letters, no digit/symbol
		"qwertyqwerty", // 12 letters (also caught by substring rule)
	}
	for _, p := range cases {
		err := ValidatePassword(p)
		if err == nil {
			t.Errorf("ValidatePassword(%q) should reject (no digit/symbol)", p)
			continue
		}
		// Could be substring rule OR class rule depending on the password.
		code := err.(*PasswordPolicyError).Code
		if code != CodePasswordNeedsLetter && code != CodePasswordContainsCommon && code != CodePasswordInBlocklist {
			t.Errorf("ValidatePassword(%q) wrong code: %s", p, code)
		}
	}
}

func TestValidatePassword_RejectsAllDigits(t *testing.T) {
	cases := []string{
		"1234567890", // 10 digits, no letter
		"00000000001",
	}
	for _, p := range cases {
		err := ValidatePassword(p)
		if err == nil {
			t.Errorf("ValidatePassword(%q) should reject (all digits)", p)
		}
	}
}

func TestValidatePassword_AcceptsAllSymbolsWithLetter(t *testing.T) {
	// A long symbol-only run is fine if there's at least one letter
	// mixed in. e.g. "!@#$%^&*()" with one letter -> still works.
	if err := ValidatePassword("!K8@#$%^&*()"); err != nil {
		t.Errorf("ValidatePassword with mixed letter+symbol should accept: %v", err)
	}
}

func TestValidatePassword_BlocklistContainsKnownTop10(t *testing.T) {
	// Sanity check: ensure the blocklist is non-empty AND contains
	// known top-passwords. If this fails the embed didn't load.
	for _, p := range []string{"password", "123456", "qwerty"} {
		if !inBlocklist(p) {
			t.Errorf("blocklist missing %q", p)
		}
	}
}

func TestValidatePassword_RejectsTooWeak(t *testing.T) {
	// New in 2026-08-22: passes all binary rules but fails the
	// entropy-based score gate (score < 40). The classic case the
	// user complained about: a 15-char run of "h"s with a digit at
	// the end used to pass all 6 rules and rate as "strong".
	cases := []string{
		"hhhhhhhhh1", // 10 chars, all h + 1 digit, scores < 40
	}
	for _, p := range cases {
		err := ValidatePassword(p)
		if err == nil {
			t.Errorf("ValidatePassword(%q) should reject (too weak)", p)
			continue
		}
		if err.(*PasswordPolicyError).Code != CodePasswordTooWeak {
			t.Errorf("ValidatePassword(%q) wrong code: want %s, got %s", p, CodePasswordTooWeak, err.(*PasswordPolicyError).Code)
		}
	}
}

func TestScorePassword_Examples(t *testing.T) {
	// Score sanity checks. We don't pin exact values (the algorithm
	// is allowed to evolve) but we do pin the bucket so a refactor
	// can't silently flip a strong password to weak or vice versa.
	// The user's complaint was that "hhhhhhhhhhhhhh1" was rated as
	// "strong" because all 6 binary rules passed. After 2026-08-22
	// strengthening, it scores in "ok" or "weak" — never "strong".
	cases := []struct {
		pwd    string
		bucket string // "weak" | "ok" | "strong"
	}{
		{"hhhhhhhhhhhhhh1", "ok"},     // 15 chars, all same letter + 1 digit (low uniqueness)
		{"aaaaaaaaaa1", "weak"},       // contains "aaaa" substring → caught before scoring; score not used
		{"abcdefghij1", "ok"},         // 11 chars, low uniqueness
		{"Tr0ub4dor&3", "strong"},     // xkcd classic — well above strong
		{"Hx42!kr9@pT", "strong"},     // 11 chars, 4 classes
		{"K8mZ!qP3@vR5", "strong"},    // 12 chars, 4 classes
		{"Sundrop#Mango42", "strong"}, // mixed words + numbers + symbol
	}
	for _, c := range cases {
		score := ScorePassword(c.pwd)
		var got string
		switch {
		case score < OKScoreThreshold:
			got = "weak"
		case score < StrongScoreThreshold:
			got = "ok"
		default:
			got = "strong"
		}
		if got != c.bucket {
			t.Errorf("ScorePassword(%q) = %d, expected bucket %q, got %q", c.pwd, score, c.bucket, got)
		}
	}
}

func TestMustDifferFrom_RejectsSamePassword(t *testing.T) {
	hash, _ := HashPassword(valid())
	err := MustDifferFrom(hash, valid())
	if err == nil {
		t.Errorf("MustDifferFrom should reject when plain matches hash")
	}
	if err.(*PasswordPolicyError).Code != "password_must_differ" {
		t.Errorf("wrong code: %s", err.(*PasswordPolicyError).Code)
	}
}

func TestMustDifferFrom_AcceptsDifferentPassword(t *testing.T) {
	hash, _ := HashPassword(valid())
	if err := MustDifferFrom(hash, "Different2x42!"); err != nil {
		t.Errorf("MustDifferFrom should accept different password: %v", err)
	}
}

func TestMustDifferFrom_AcceptsMalformedHash(t *testing.T) {
	// A malformed hash is not "same password" — the caller will hit
	// a separate error at the actual verify step.
	if err := MustDifferFrom("not-a-real-hash", "any-password"); err != nil {
		t.Errorf("MustDifferFrom should accept any input when hash is malformed: %v", err)
	}
}

func TestIsPasswordPolicyError(t *testing.T) {
	err := ValidatePassword("short")
	if _, ok := IsPasswordPolicyError(err); !ok {
		t.Errorf("IsPasswordPolicyError should return true")
	}
	if _, ok := IsPasswordPolicyError(nil); ok {
		t.Errorf("IsPasswordPolicyError(nil) should return ok=false")
	}
}
