package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("hunter22-strong")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if !CheckPassword(hash, "hunter22-strong") {
		t.Fatal("password should match")
	}
	if CheckPassword(hash, "wrong") {
		t.Fatal("wrong password should not match")
	}
}

func TestHashPassword_TooShort(t *testing.T) {
	if _, err := HashPassword("short"); err == nil {
		t.Fatal("short password should fail")
	}
}

func TestIssueAndVerify(t *testing.T) {
	iss := NewIssuer("test-secret-32-bytes-or-more-please", 15*time.Minute)
	uid := uuid.New()
	tid := uuid.New()
	tok, err := iss.Issue(uid, tid, "a@b.com", "admin", false)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	c, err := iss.Verify(tok)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if c.UserID != uid || c.TenantID != tid || c.Email != "a@b.com" || c.Role != "admin" {
		t.Fatalf("claims mismatch: %+v", c)
	}
}

func TestVerify_BadToken(t *testing.T) {
	iss := NewIssuer("test-secret-32-bytes-or-more-please", 15*time.Minute)
	if _, err := iss.Verify("not-a-jwt"); err == nil {
		t.Fatal("bad token should fail")
	}
}

func TestVerify_DifferentSecret(t *testing.T) {
	iss1 := NewIssuer("secret-one-32-bytes-long-secret-xxxxx", 15*time.Minute)
	iss2 := NewIssuer("secret-two-32-bytes-long-secret-xxxxx", 15*time.Minute)
	uid := uuid.New()
	tid := uuid.New()
	tok, _ := iss1.Issue(uid, tid, "a@b.com", "admin", false)
	if _, err := iss2.Verify(tok); err == nil {
		t.Fatal("verify with different secret should fail")
	}
}
