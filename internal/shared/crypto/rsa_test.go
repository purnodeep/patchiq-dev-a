package crypto_test

import (
	"crypto/rsa"
	"testing"

	"github.com/skenzeriq/patchiq/internal/shared/crypto"
)

func TestGenerateKeyPair(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		wantBit int
	}{
		{name: "generates 2048-bit key", wantBit: 2048},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			key, err := crypto.GenerateKeyPair()
			if err != nil {
				t.Fatalf("GenerateKeyPair() error = %v", err)
			}
			if key == nil {
				t.Fatal("GenerateKeyPair() returned nil key")
				return
			}
			if key.N.BitLen() != tc.wantBit {
				t.Errorf("key bit length = %d, want %d", key.N.BitLen(), tc.wantBit)
			}
		})
	}
}

func TestSignAndVerify(t *testing.T) {
	t.Parallel()

	key, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	tests := []struct {
		name    string
		payload []byte
	}{
		{name: "short payload", payload: []byte("hello")},
		{name: "empty payload", payload: []byte{}},
		{name: "long payload", payload: make([]byte, 10000)},
		{name: "json-like payload", payload: []byte(`{"license":"test","seats":10}`)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sig, err := crypto.SignPayload(key, tc.payload)
			if err != nil {
				t.Fatalf("SignPayload() error = %v", err)
			}
			if len(sig) == 0 {
				t.Fatal("SignPayload() returned empty signature")
			}

			if err := crypto.VerifySignature(&key.PublicKey, tc.payload, sig); err != nil {
				t.Errorf("VerifySignature() error = %v", err)
			}
		})
	}
}

func TestVerifyRejectsTamperedPayload(t *testing.T) {
	t.Parallel()

	key, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	original := []byte("original payload")
	sig, err := crypto.SignPayload(key, original)
	if err != nil {
		t.Fatalf("SignPayload() error = %v", err)
	}

	tests := []struct {
		name    string
		payload []byte
		sig     []byte
	}{
		{
			name:    "tampered payload",
			payload: []byte("tampered payload"),
			sig:     sig,
		},
		{
			name:    "truncated signature",
			payload: original,
			sig:     sig[:len(sig)-1],
		},
		{
			name:    "wrong key",
			payload: original,
			sig: func() []byte {
				other, _ := crypto.GenerateKeyPair()
				otherSig, _ := crypto.SignPayload(other, original)
				return otherSig
			}(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := crypto.VerifySignature(&key.PublicKey, tc.payload, tc.sig)
			if err == nil {
				t.Error("VerifySignature() expected error for tampered data, got nil")
			}
		})
	}
}

func TestPEMRoundTrip(t *testing.T) {
	t.Parallel()

	key, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	t.Run("private key round-trip", func(t *testing.T) {
		t.Parallel()

		pem := crypto.EncodePrivateKeyPEM(key)
		if len(pem) == 0 {
			t.Fatal("EncodePrivateKeyPEM() returned empty bytes")
		}

		decoded, err := crypto.DecodePrivateKeyPEM(pem)
		if err != nil {
			t.Fatalf("DecodePrivateKeyPEM() error = %v", err)
		}
		if !key.Equal(decoded) {
			t.Error("decoded private key does not equal original")
		}
	})

	t.Run("public key round-trip", func(t *testing.T) {
		t.Parallel()

		pem, err := crypto.EncodePublicKeyPEM(&key.PublicKey)
		if err != nil {
			t.Fatalf("EncodePublicKeyPEM() error = %v", err)
		}
		if len(pem) == 0 {
			t.Fatal("EncodePublicKeyPEM() returned empty bytes")
		}

		decoded, err := crypto.DecodePublicKeyPEM(pem)
		if err != nil {
			t.Fatalf("DecodePublicKeyPEM() error = %v", err)
		}
		if !key.PublicKey.Equal(decoded) {
			t.Error("decoded public key does not equal original")
		}
	})

	t.Run("round-tripped keys can sign and verify", func(t *testing.T) {
		t.Parallel()

		privPEM := crypto.EncodePrivateKeyPEM(key)
		privKey, err := crypto.DecodePrivateKeyPEM(privPEM)
		if err != nil {
			t.Fatalf("DecodePrivateKeyPEM() error = %v", err)
		}

		pubPEM, err := crypto.EncodePublicKeyPEM(&key.PublicKey)
		if err != nil {
			t.Fatalf("EncodePublicKeyPEM() error = %v", err)
		}
		pubKey, err := crypto.DecodePublicKeyPEM(pubPEM)
		if err != nil {
			t.Fatalf("DecodePublicKeyPEM() error = %v", err)
		}

		payload := []byte("license payload after round-trip")
		sig, err := crypto.SignPayload(privKey, payload)
		if err != nil {
			t.Fatalf("SignPayload() error = %v", err)
		}
		if err := crypto.VerifySignature(pubKey, payload, sig); err != nil {
			t.Errorf("VerifySignature() with round-tripped keys error = %v", err)
		}
	})

	t.Run("decode invalid private PEM", func(t *testing.T) {
		t.Parallel()

		if _, err := crypto.DecodePrivateKeyPEM([]byte("not a pem")); err == nil {
			t.Error("DecodePrivateKeyPEM() expected error for invalid PEM, got nil")
		}
	})

	t.Run("decode invalid public PEM", func(t *testing.T) {
		t.Parallel()

		if _, err := crypto.DecodePublicKeyPEM([]byte("not a pem")); err == nil {
			t.Error("DecodePublicKeyPEM() expected error for invalid PEM, got nil")
		}
	})
}

// Ensure the function signatures match expected types (compile-time check).
var (
	_ func() (*rsa.PrivateKey, error)               = crypto.GenerateKeyPair
	_ func(*rsa.PrivateKey, []byte) ([]byte, error) = crypto.SignPayload
	_ func(*rsa.PublicKey, []byte, []byte) error    = crypto.VerifySignature
	_ func(*rsa.PrivateKey) []byte                  = crypto.EncodePrivateKeyPEM
	_ func([]byte) (*rsa.PrivateKey, error)         = crypto.DecodePrivateKeyPEM
	_ func(*rsa.PublicKey) ([]byte, error)          = crypto.EncodePublicKeyPEM
	_ func([]byte) (*rsa.PublicKey, error)          = crypto.DecodePublicKeyPEM
)
