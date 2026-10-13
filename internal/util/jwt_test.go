package util

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-secret"

func TestGenerateAndParseToken_RoundTrip(t *testing.T) {
	token, err := GenerateToken(42, "admin", testSecret)
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}

	claims, err := ParseToken(token, testSecret)
	if err != nil {
		t.Fatalf("ParseToken returned error: %v", err)
	}
	if claims.UserID != 42 {
		t.Errorf("UserID = %d, want 42", claims.UserID)
	}
	if claims.Role != "admin" {
		t.Errorf("Role = %q, want %q", claims.Role, "admin")
	}
	if claims.ExpiresAt == nil || !claims.ExpiresAt.After(time.Now()) {
		t.Error("token expiry must be set in the future")
	}
}

func TestParseToken_Rejects(t *testing.T) {
	claims := Claims{
		UserID: 1,
		Role:   "user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}

	expired := claims
	expired.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Hour))

	sign := func(t *testing.T, method jwt.SigningMethod, c Claims, key any) string {
		t.Helper()
		token, err := jwt.NewWithClaims(method, c).SignedString(key)
		if err != nil {
			t.Fatalf("failed to sign test token: %v", err)
		}
		return token
	}

	validForOtherSecret, err := GenerateToken(1, "user", "another-secret")
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}

	tests := []struct {
		name  string
		token string
	}{
		{"empty string", ""},
		{"garbage", "not-a-token"},
		{"signed with a different secret", validForOtherSecret},
		{"expired", sign(t, jwt.SigningMethodHS256, expired, []byte(testSecret))},
		{"alg none", sign(t, jwt.SigningMethodNone, claims, jwt.UnsafeAllowNoneSignatureType)},
		{"unexpected HMAC algorithm", sign(t, jwt.SigningMethodHS384, claims, []byte(testSecret))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParseToken(tt.token, testSecret); err == nil {
				t.Fatal("ParseToken accepted a token it should reject")
			}
		})
	}
}
