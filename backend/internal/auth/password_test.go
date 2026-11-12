package auth

import (
	"strings"
	"testing"
)

func TestHashIsNotThePassword(t *testing.T) { // T5
	hash, err := HashPassword("correct horse")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if strings.Contains(hash, "correct horse") {
		t.Fatalf("the stored value contains the password: %q", hash)
	}
}

func TestCorrectPasswordMatchesItsHash(t *testing.T) {
	hash, err := HashPassword("correct horse")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if !CheckPassword(hash, "correct horse") {
		t.Fatal("the right password was refused")
	}
}

func TestWrongPasswordDoesNotMatch(t *testing.T) {
	hash, err := HashPassword("correct horse")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if CheckPassword(hash, "wrong horse") {
		t.Fatal("a wrong password was accepted")
	}
}

func TestSamePasswordHashesDifferentlyEachTime(t *testing.T) {
	// Each hash has its own random salt, so two people with one password differ.
	a, _ := HashPassword("correct horse")
	b, _ := HashPassword("correct horse")
	if a == b {
		t.Fatal("two hashes of the same password are equal")
	}
}

func TestBrokenHashNeverMatches(t *testing.T) {
	for _, hash := range []string{"", "plain-text", "pbkdf2-sha256$x$y$z", "pbkdf2-sha256$1$$"} {
		if CheckPassword(hash, "anything") {
			t.Fatalf("broken hash %q accepted a password", hash)
		}
	}
}

func TestSessionTokenIsStoredOnlyAsAHash(t *testing.T) {
	token, hash, err := NewSessionToken()
	if err != nil {
		t.Fatalf("new token: %v", err)
	}
	if len(token) < 32 {
		t.Fatalf("token too short: %d characters", len(token))
	}
	if hash == token || strings.Contains(hash, token) {
		t.Fatal("the stored value contains the token")
	}
	if HashSessionToken(token) != hash {
		t.Fatal("hashing the token again gives a different value")
	}
}
