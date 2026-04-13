package auth_test

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/skenzeriq/patchiq/internal/hub/auth"
)

// TestInitSigningKey verifies that InitSigningKey generates or preserves signing keys.
func TestInitSigningKey(t *testing.T) {
	t.Run("generates 32-byte key when empty", func(t *testing.T) {
		cfg := &auth.SessionConfig{}
		cfg.InitSigningKey()

		if len(cfg.SigningKey) != 32 {
			t.Errorf("SigningKey length = %d, want 32", len(cfg.SigningKey))
		}
	})

	t.Run("does not overwrite existing key", func(t *testing.T) {
		existing := []byte("my-32-byte-secret-key-1234567890")
		cfg := &auth.SessionConfig{
			SigningKey: existing,
		}
		cfg.InitSigningKey()

		if !bytes.Equal(cfg.SigningKey, existing) {
			t.Errorf("SigningKey was overwritten; got %x, want %x", cfg.SigningKey, existing)
		}
	})

	t.Run("generates unique keys on each call", func(t *testing.T) {
		cfg1 := &auth.SessionConfig{}
		cfg2 := &auth.SessionConfig{}
		cfg1.InitSigningKey()
		cfg2.InitSigningKey()

		if bytes.Equal(cfg1.SigningKey, cfg2.SigningKey) {
			t.Error("two separate InitSigningKey calls produced identical keys (astronomically unlikely)")
		}
	})
}

// TestMintJWT exercises the exported MintJWT helper (or internal via test package trick).
// We expose mintJWT indirectly through a test helper exported only during tests — but
// since mintJWT is unexported, we call it via auth.MintJWTForTest which the package
// exposes for testing purposes.
func TestMintJWT(t *testing.T) {
	key := []byte("test-signing-key-32-bytes-padded!")
	sub := "user@example.com"
	tenantID := "tenant-uuid-1234"
	email := "user@example.com"
	name := "Test User"
	ttl := 1 * time.Hour

	type jwtClaims struct {
		Sub      string `json:"sub"`
		TenantID string `json:"tenant_id"`
		Email    string `json:"email"`
		Name     string `json:"name"`
		Iss      string `json:"iss"`
		Iat      int64  `json:"iat"`
		Exp      int64  `json:"exp"`
	}

	tests := []struct {
		name    string
		key     []byte
		sub     string
		tenant  string
		email   string
		uname   string
		ttl     time.Duration
		wantErr bool
	}{
		{
			name:   "valid token",
			key:    key,
			sub:    sub,
			tenant: tenantID,
			email:  email,
			uname:  name,
			ttl:    ttl,
		},
		{
			name:   "short TTL still produces valid token",
			key:    key,
			sub:    "admin@hub.io",
			tenant: "other-tenant",
			email:  "admin@hub.io",
			uname:  "Admin",
			ttl:    1 * time.Minute,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			token, err := auth.MintJWTForTest(tc.key, tc.sub, tc.tenant, tc.email, tc.uname, tc.ttl)
			if (err != nil) != tc.wantErr {
				t.Fatalf("MintJWTForTest() error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}

			// Must be a 3-part JWT.
			parts := strings.Split(token, ".")
			if len(parts) != 3 {
				t.Fatalf("token has %d parts, want 3", len(parts))
			}

			// Header must decode to HS256.
			headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
			if err != nil {
				t.Fatalf("header base64 decode failed: %v", err)
			}
			var header map[string]string
			if err := json.Unmarshal(headerJSON, &header); err != nil {
				t.Fatalf("header JSON unmarshal failed: %v", err)
			}
			if header["alg"] != "HS256" {
				t.Errorf("alg = %q, want %q", header["alg"], "HS256")
			}
			if header["typ"] != "JWT" {
				t.Errorf("typ = %q, want %q", header["typ"], "JWT")
			}

			// Claims must contain expected fields.
			claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
			if err != nil {
				t.Fatalf("claims base64 decode failed: %v", err)
			}
			var claims jwtClaims
			if err := json.Unmarshal(claimsJSON, &claims); err != nil {
				t.Fatalf("claims JSON unmarshal failed: %v", err)
			}

			if claims.Sub != tc.sub {
				t.Errorf("sub = %q, want %q", claims.Sub, tc.sub)
			}
			if claims.TenantID != tc.tenant {
				t.Errorf("tenant_id = %q, want %q", claims.TenantID, tc.tenant)
			}
			if claims.Email != tc.email {
				t.Errorf("email = %q, want %q", claims.Email, tc.email)
			}
			if claims.Name != tc.uname {
				t.Errorf("name = %q, want %q", claims.Name, tc.uname)
			}
			if claims.Iss != "patchiq-hub" {
				t.Errorf("iss = %q, want %q", claims.Iss, "patchiq-hub")
			}
			if claims.Iat <= 0 {
				t.Errorf("iat = %d, want > 0", claims.Iat)
			}
			if claims.Exp <= claims.Iat {
				t.Errorf("exp (%d) must be after iat (%d)", claims.Exp, claims.Iat)
			}

			// HMAC signature must be valid.
			sigInput := parts[0] + "." + parts[1]
			mac := hmac.New(sha256.New, tc.key)
			mac.Write([]byte(sigInput))
			wantSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
			if parts[2] != wantSig {
				t.Errorf("signature mismatch\ngot:  %s\nwant: %s", parts[2], wantSig)
			}
		})
	}
}

func TestMintJWT_NoClaimInjection(t *testing.T) {
	key := []byte("test-key-for-injection-test-pad!")
	maliciousName := `","role":"superadmin`
	token, err := auth.MintJWTForTest(key, "user-1", "tenant-1", "evil@example.com", maliciousName, time.Hour)
	if err != nil {
		t.Fatalf("MintJWTForTest: %v", err)
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("token has %d parts, want 3", len(parts))
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}

	var claims map[string]any
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}

	if claims["name"] != maliciousName {
		t.Errorf("name = %q, want %q", claims["name"], maliciousName)
	}
	if _, hasRole := claims["role"]; hasRole {
		t.Error("JWT claim injection: attacker injected a 'role' claim")
	}
}

// TestMintJWT_TenantInClaims verifies tenant_id is correctly embedded in claims.
func TestMintJWT_TenantInClaims(t *testing.T) {
	key := []byte("another-32-byte-test-signing-key")
	cases := []struct {
		tenantID string
	}{
		{"00000000-0000-0000-0000-000000000001"},
		{"tenant-abc-xyz"},
		{""},
	}

	for _, tc := range cases {
		t.Run("tenant_id="+tc.tenantID, func(t *testing.T) {
			token, err := auth.MintJWTForTest(key, "u1", tc.tenantID, "u1@x.com", "U One", 5*time.Minute)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			parts := strings.Split(token, ".")
			claimsJSON, _ := base64.RawURLEncoding.DecodeString(parts[1])

			var claims map[string]interface{}
			if err := json.Unmarshal(claimsJSON, &claims); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}

			got, ok := claims["tenant_id"]
			if !ok {
				t.Fatal("tenant_id claim missing")
			}
			gotStr, _ := got.(string)
			if gotStr != tc.tenantID {
				t.Errorf("tenant_id = %q, want %q", got, tc.tenantID)
			}
		})
	}
}
