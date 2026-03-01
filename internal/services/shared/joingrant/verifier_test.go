package joingrant

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
)

func TestLoadConfigFromEnv(t *testing.T) {
	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_ISSUER", "issuer-1")
	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_AUDIENCE", "aud-1")
	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_PUBLIC_KEY", base64.RawStdEncoding.EncodeToString(publicKey))

	now := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	cfg, err := LoadConfigFromEnv(func() time.Time { return now })
	if err != nil {
		t.Fatalf("LoadConfigFromEnv: %v", err)
	}
	if cfg.Issuer != "issuer-1" || cfg.Audience != "aud-1" {
		t.Fatalf("unexpected config values: %+v", cfg)
	}
	if cfg.Now == nil || !cfg.Now().Equal(now) {
		t.Fatalf("cfg.Now mismatch")
	}
}

func TestEnvVerifier_NotConfigured(t *testing.T) {
	_, err := (EnvVerifier{}).Validate("token", Expectation{})
	if err == nil || !errors.Is(err, ErrVerifierNotConfigured) {
		t.Fatalf("expected ErrVerifierNotConfigured, got %v", err)
	}
}

func TestValidate_Success(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	now := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	token := signGrant(t, privateKey, jwt.MapClaims{
		"iss":         "issuer-1",
		"aud":         "aud-1",
		"exp":         now.Add(5 * time.Minute).Unix(),
		"nbf":         now.Add(-1 * time.Minute).Unix(),
		"iat":         now.Add(-1 * time.Minute).Unix(),
		"jti":         "jti-1",
		"campaign_id": "camp-1",
		"invite_id":   "inv-1",
		"user_id":     "user-1",
	})

	claims, err := Validate(token, Expectation{
		CampaignID: "camp-1",
		InviteID:   "inv-1",
		UserID:     "user-1",
	}, Config{
		Issuer:   "issuer-1",
		Audience: "aud-1",
		Key:      publicKey,
		Now:      func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if claims.JWTID != "jti-1" {
		t.Fatalf("JWTID = %q, want jti-1", claims.JWTID)
	}
}

func TestValidate_Expired(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	now := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	token := signGrant(t, privateKey, jwt.MapClaims{
		"iss":         "issuer-1",
		"aud":         "aud-1",
		"exp":         now.Add(-1 * time.Minute).Unix(),
		"jti":         "jti-1",
		"campaign_id": "camp-1",
		"invite_id":   "inv-1",
		"user_id":     "user-1",
	})

	_, err = Validate(token, Expectation{
		CampaignID: "camp-1",
		InviteID:   "inv-1",
		UserID:     "user-1",
	}, Config{
		Issuer:   "issuer-1",
		Audience: "aud-1",
		Key:      publicKey,
		Now:      func() time.Time { return now },
	})
	if apperrors.GetCode(err) != apperrors.CodeInviteJoinGrantExpired {
		t.Fatalf("code = %s, want %s (err=%v)", apperrors.GetCode(err), apperrors.CodeInviteJoinGrantExpired, err)
	}
}

func TestStaticVerifier_Validate(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	now := time.Date(2026, 2, 1, 11, 0, 0, 0, time.UTC)
	token := signGrant(t, privateKey, jwt.MapClaims{
		"iss":         "issuer-1",
		"aud":         "aud-1",
		"exp":         now.Add(5 * time.Minute).Unix(),
		"jti":         "jti-1",
		"campaign_id": "camp-1",
		"invite_id":   "inv-1",
		"user_id":     "user-1",
	})

	_, err = (StaticVerifier{Config: Config{
		Issuer:   "issuer-1",
		Audience: "aud-1",
		Key:      publicKey,
		Now:      func() time.Time { return now },
	}}).Validate(token, Expectation{
		CampaignID: "camp-1",
		InviteID:   "inv-1",
		UserID:     "user-1",
	})
	if err != nil {
		t.Fatalf("StaticVerifier.Validate: %v", err)
	}
}

func TestEnvVerifier_Validate(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	now := time.Date(2026, 2, 2, 9, 0, 0, 0, time.UTC)
	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_ISSUER", "issuer-1")
	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_AUDIENCE", "aud-1")
	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_PUBLIC_KEY", base64.RawStdEncoding.EncodeToString(publicKey))

	token := signGrant(t, privateKey, jwt.MapClaims{
		"iss":         "issuer-1",
		"aud":         "aud-1",
		"exp":         now.Add(5 * time.Minute).Unix(),
		"jti":         "jti-1",
		"campaign_id": "camp-1",
		"invite_id":   "inv-1",
		"user_id":     "user-1",
	})
	_, err = (EnvVerifier{Now: func() time.Time { return now }}).Validate(token, Expectation{
		CampaignID: "camp-1",
		InviteID:   "inv-1",
		UserID:     "user-1",
	})
	if err != nil {
		t.Fatalf("EnvVerifier.Validate: %v", err)
	}
}

func TestLoadConfigFromEnv_ValidationErrors(t *testing.T) {
	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	tests := []struct {
		name      string
		issuer    string
		audience  string
		publicKey string
	}{
		{name: "missing issuer", audience: "aud-1", publicKey: base64.RawStdEncoding.EncodeToString(publicKey)},
		{name: "missing audience", issuer: "issuer-1", publicKey: base64.RawStdEncoding.EncodeToString(publicKey)},
		{name: "missing key", issuer: "issuer-1", audience: "aud-1"},
		{name: "invalid base64", issuer: "issuer-1", audience: "aud-1", publicKey: "***"},
		{name: "wrong key size", issuer: "issuer-1", audience: "aud-1", publicKey: base64.RawStdEncoding.EncodeToString([]byte("short"))},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("FRACTURING_SPACE_JOIN_GRANT_ISSUER", tc.issuer)
			t.Setenv("FRACTURING_SPACE_JOIN_GRANT_AUDIENCE", tc.audience)
			t.Setenv("FRACTURING_SPACE_JOIN_GRANT_PUBLIC_KEY", tc.publicKey)
			if _, err := LoadConfigFromEnv(nil); err == nil {
				t.Fatalf("expected error")
			}
		})
	}
}

func TestLoadConfigFromEnv_DefaultNow(t *testing.T) {
	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_ISSUER", "issuer-1")
	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_AUDIENCE", "aud-1")
	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_PUBLIC_KEY", base64.RawStdEncoding.EncodeToString(publicKey))

	cfg, err := LoadConfigFromEnv(nil)
	if err != nil {
		t.Fatalf("LoadConfigFromEnv: %v", err)
	}
	if cfg.Now == nil {
		t.Fatalf("cfg.Now should default to time.Now")
	}
}

func TestValidate_RejectsInvalidConfigurationsAndClaims(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	otherPublic, otherPrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey other keypair: %v", err)
	}
	now := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	baseClaims := jwt.MapClaims{
		"iss":         "issuer-1",
		"aud":         "aud-1",
		"exp":         now.Add(10 * time.Minute).Unix(),
		"jti":         "jti-1",
		"campaign_id": "camp-1",
		"invite_id":   "inv-1",
		"user_id":     "user-1",
	}
	baseCfg := Config{
		Issuer:   "issuer-1",
		Audience: "aud-1",
		Key:      publicKey,
		Now:      func() time.Time { return now },
	}
	expected := Expectation{CampaignID: "camp-1", InviteID: "inv-1", UserID: "user-1"}

	t.Run("empty grant", func(t *testing.T) {
		_, err := Validate("", expected, baseCfg)
		if apperrors.GetCode(err) != apperrors.CodeInviteJoinGrantInvalid {
			t.Fatalf("code = %s, want %s", apperrors.GetCode(err), apperrors.CodeInviteJoinGrantInvalid)
		}
	})

	t.Run("missing verifier config", func(t *testing.T) {
		token := signGrant(t, privateKey, baseClaims)
		_, err := Validate(token, expected, Config{})
		if err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("signature invalid", func(t *testing.T) {
		token := signGrant(t, otherPrivate, baseClaims)
		_, err := Validate(token, expected, baseCfg)
		if apperrors.GetCode(err) != apperrors.CodeInviteJoinGrantInvalid {
			t.Fatalf("code = %s, want %s", apperrors.GetCode(err), apperrors.CodeInviteJoinGrantInvalid)
		}
	})

	t.Run("issuer mismatch", func(t *testing.T) {
		claims := cloneClaims(baseClaims)
		claims["iss"] = "issuer-2"
		token := signGrant(t, privateKey, claims)
		_, err := Validate(token, expected, baseCfg)
		if apperrors.GetCode(err) != apperrors.CodeInviteJoinGrantMismatch {
			t.Fatalf("code = %s, want %s", apperrors.GetCode(err), apperrors.CodeInviteJoinGrantMismatch)
		}
	})

	t.Run("audience mismatch", func(t *testing.T) {
		claims := cloneClaims(baseClaims)
		claims["aud"] = "aud-2"
		token := signGrant(t, privateKey, claims)
		_, err := Validate(token, expected, baseCfg)
		if apperrors.GetCode(err) != apperrors.CodeInviteJoinGrantMismatch {
			t.Fatalf("code = %s, want %s", apperrors.GetCode(err), apperrors.CodeInviteJoinGrantMismatch)
		}
	})

	t.Run("missing jti", func(t *testing.T) {
		claims := cloneClaims(baseClaims)
		delete(claims, "jti")
		token := signGrant(t, privateKey, claims)
		_, err := Validate(token, expected, baseCfg)
		if apperrors.GetCode(err) != apperrors.CodeInviteJoinGrantInvalid {
			t.Fatalf("code = %s, want %s", apperrors.GetCode(err), apperrors.CodeInviteJoinGrantInvalid)
		}
	})

	t.Run("missing exp", func(t *testing.T) {
		claims := cloneClaims(baseClaims)
		delete(claims, "exp")
		token := signGrant(t, privateKey, claims)
		_, err := Validate(token, expected, baseCfg)
		if apperrors.GetCode(err) != apperrors.CodeInviteJoinGrantInvalid {
			t.Fatalf("code = %s, want %s", apperrors.GetCode(err), apperrors.CodeInviteJoinGrantInvalid)
		}
	})

	t.Run("not before in future", func(t *testing.T) {
		claims := cloneClaims(baseClaims)
		claims["nbf"] = now.Add(1 * time.Minute).Unix()
		token := signGrant(t, privateKey, claims)
		_, err := Validate(token, expected, baseCfg)
		if apperrors.GetCode(err) != apperrors.CodeInviteJoinGrantInvalid {
			t.Fatalf("code = %s, want %s", apperrors.GetCode(err), apperrors.CodeInviteJoinGrantInvalid)
		}
	})

	t.Run("campaign mismatch", func(t *testing.T) {
		claims := cloneClaims(baseClaims)
		claims["campaign_id"] = "camp-2"
		token := signGrant(t, privateKey, claims)
		_, err := Validate(token, expected, baseCfg)
		if apperrors.GetCode(err) != apperrors.CodeInviteJoinGrantMismatch {
			t.Fatalf("code = %s, want %s", apperrors.GetCode(err), apperrors.CodeInviteJoinGrantMismatch)
		}
	})

	t.Run("invite mismatch", func(t *testing.T) {
		claims := cloneClaims(baseClaims)
		claims["invite_id"] = "inv-2"
		token := signGrant(t, privateKey, claims)
		_, err := Validate(token, expected, baseCfg)
		if apperrors.GetCode(err) != apperrors.CodeInviteJoinGrantMismatch {
			t.Fatalf("code = %s, want %s", apperrors.GetCode(err), apperrors.CodeInviteJoinGrantMismatch)
		}
	})

	t.Run("user mismatch", func(t *testing.T) {
		claims := cloneClaims(baseClaims)
		claims["user_id"] = "user-2"
		token := signGrant(t, privateKey, claims)
		_, err := Validate(token, expected, baseCfg)
		if apperrors.GetCode(err) != apperrors.CodeInviteJoinGrantMismatch {
			t.Fatalf("code = %s, want %s", apperrors.GetCode(err), apperrors.CodeInviteJoinGrantMismatch)
		}
	})

	_ = otherPublic
}

func TestMapJWTError(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "signature invalid", err: jwt.ErrTokenSignatureInvalid},
		{name: "ed25519 invalid", err: jwt.ErrEd25519Verification},
		{name: "unverifiable", err: jwt.ErrTokenUnverifiable},
		{name: "fallback", err: errors.New("boom")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mapped := mapJWTError(tc.err)
			if apperrors.GetCode(mapped) != apperrors.CodeInviteJoinGrantInvalid {
				t.Fatalf("mapped code = %s, want %s", apperrors.GetCode(mapped), apperrors.CodeInviteJoinGrantInvalid)
			}
		})
	}
}

func TestAudienceContainsAndDecodeBase64(t *testing.T) {
	if audienceContains(jwt.ClaimStrings{"aud-1", "aud-2"}, "aud-3") {
		t.Fatalf("audienceContains should be false for missing audience")
	}
	if !audienceContains(jwt.ClaimStrings{"aud-1", "aud-2"}, "aud-2") {
		t.Fatalf("audienceContains should be true when audience exists")
	}

	if _, err := decodeBase64(""); err == nil {
		t.Fatalf("decodeBase64 should reject empty input")
	}
	data := []byte("01234567890123456789012345678901")
	raw := base64.RawStdEncoding.EncodeToString(data)
	std := base64.StdEncoding.EncodeToString(data)

	rawDecoded, err := decodeBase64(raw)
	if err != nil {
		t.Fatalf("decodeBase64 raw: %v", err)
	}
	if string(rawDecoded) != string(data) {
		t.Fatalf("raw decoded mismatch")
	}

	stdDecoded, err := decodeBase64(std)
	if err != nil {
		t.Fatalf("decodeBase64 std: %v", err)
	}
	if string(stdDecoded) != string(data) {
		t.Fatalf("std decoded mismatch")
	}
}

func cloneClaims(in jwt.MapClaims) jwt.MapClaims {
	out := make(jwt.MapClaims, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func signGrant(t *testing.T, key ed25519.PrivateKey, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("SignedString: %v", err)
	}
	return signed
}
