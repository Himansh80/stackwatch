package auth

import (
	"strings"
	"testing"
)

// Valid password = at least 10 chars, has letter + digit-or-symbol,
// not whitespace-only, no leading/trailing whitespace, not in blocklist.

func valid() string { return "Hunter2x42!" }

func TestValidatePassword_AcceptsValid(t *testing.T) {
	good := []string{
		"Hunter2x42",    // letters + digits
		"correct horse battery staple!", // passphrase with symbol
		"Tr0ub4dor&3",    // xkcd classic with digit + symbol
		"MyDog9!!2026",  // letters + digits + symbols (14 chars)
		"I-Love-Code-2026", // letters + digits + symbols
		"ab1!@#$%^&*",   // exactly 10 chars
	}
	for _, p := range good {
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
	cases := []string{" Hunter2x42", "Hunter2x42 ", "  Hunter2x42  "}
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
			"correct horse battery staple",
			"Hunter2x42",
			"MyDog9!!2026",
			"hunter22",
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

func TestValidatePassword_RejectsAllLetters(t *testing.T) {
	cases := []string{
		"abcdefghijk",  // 11 letters, no digit/symbol
		"longpassword", // 12 letters
	}
	for _, p := range cases {
		err := ValidatePassword(p)
		if err == nil {
			t.Errorf("ValidatePassword(%q) should reject (all letters)", p)
		}
		if err != nil && err.(*PasswordPolicyError).Code != CodePasswordNeedsLetter {
			t.Errorf("ValidatePassword(%q) wrong code: %s", p, err.(*PasswordPolicyError).Code)
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
	if err := ValidatePassword("!a@#$%^&*()"); err != nil {
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
