package oauth

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"regexp"
)

// ValidCodeVerifierPattern matches valid PKCE code_verifier format.
var ValidCodeVerifierPattern = regexp.MustCompile(`^[A-Za-z0-9\-._~]{43,128}$`)

// ValidatePKCE validates the code_verifier against the stored code_challenge.
func ValidatePKCE(codeVerifier, codeChallenge, codeChallengeMethod string) bool {
	if codeChallengeMethod != "S256" {
		return false
	}
	if !ValidCodeVerifierPattern.MatchString(codeVerifier) {
		return false
	}
	computed := ComputeS256Challenge(codeVerifier)
	return subtle.ConstantTimeCompare([]byte(computed), []byte(codeChallenge)) == 1
}

// ComputeS256Challenge computes the S256 code challenge from a verifier.
func ComputeS256Challenge(codeVerifier string) string {
	hash := sha256.Sum256([]byte(codeVerifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// ValidateCodeChallenge checks if a code_challenge is properly formatted.
func ValidateCodeChallenge(codeChallenge string) bool {
	if len(codeChallenge) != 43 {
		return false
	}
	for _, c := range codeChallenge {
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
			(c >= '0' && c <= '9') || c == '-' || c == '_') {
			return false
		}
	}
	return true
}
