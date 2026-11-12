package auth

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

// Passwords are stored as "pbkdf2-sha256$<rounds>$<salt>$<key>". PBKDF2 is in the
// Go standard library, so hashing needs no outside dependency.
const (
	hashPrefix = "pbkdf2-sha256"
	hashRounds = 600_000 // slow on purpose: makes guessing a stolen hash expensive
	saltBytes  = 16
	keyBytes   = 32
)

// HashPassword turns a password into the value that is stored. The password
// itself is never stored or logged.
func HashPassword(password string) (string, error) {
	salt := make([]byte, saltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("make salt: %w", err)
	}
	key, err := pbkdf2.Key(sha256.New, password, salt, hashRounds, keyBytes)
	if err != nil {
		return "", fmt.Errorf("derive key: %w", err)
	}
	return fmt.Sprintf("%s$%d$%s$%s", hashPrefix, hashRounds,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key)), nil
}

// CheckPassword reports whether the password matches the stored hash.
// A hash it cannot read never matches.
func CheckPassword(stored, password string) bool {
	parts := strings.Split(stored, "$")
	if len(parts) != 4 || parts[0] != hashPrefix {
		return false
	}
	rounds, err := strconv.Atoi(parts[1])
	if err != nil || rounds < 1 {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil || len(salt) == 0 {
		return false
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil || len(want) == 0 {
		return false
	}
	got, err := pbkdf2.Key(sha256.New, password, salt, rounds, len(want))
	if err != nil {
		return false
	}
	// Constant-time compare, so the time taken does not leak how close a guess was.
	return subtle.ConstantTimeCompare(got, want) == 1
}

// NewSessionToken makes a random session token. The token goes into the cookie;
// only its hash goes into the database.
func NewSessionToken() (token, hash string, err error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", fmt.Errorf("make session token: %w", err)
	}
	token = hex.EncodeToString(raw)
	return token, HashSessionToken(token), nil
}

// HashSessionToken gives the value under which a session is stored and looked up.
func HashSessionToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
