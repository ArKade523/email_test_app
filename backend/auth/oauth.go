package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

// GenerateCodeVerifier creates a code verifier for PKCE
func GenerateCodeVerifier() (string, error) {
	codeVerifier := make([]byte, 32)
	_, err := rand.Read(codeVerifier)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(codeVerifier), nil
}

// GenerateCodeChallenge creates a code challenge from the code verifier
func GenerateCodeChallenge(verifier string) string {
	sha := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sha[:])
}
