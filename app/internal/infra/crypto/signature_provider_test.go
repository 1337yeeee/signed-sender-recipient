package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"testing"
)

func TestECDSASHA256ProviderSignAndVerify(t *testing.T) {
	provider := NewECDSASHA256Provider()
	privateKeyPEM, publicKeyPEM := generateECDSAKeyPairPEM(t)
	message := []byte("hello signed world")

	signature, err := provider.Sign(message, privateKeyPEM)
	if err != nil {
		t.Fatalf("sign message: %v", err)
	}

	if err := provider.Verify(message, signature, publicKeyPEM); err != nil {
		t.Fatalf("verify valid signature: %v", err)
	}
}

func TestECDSASHA256ProviderVerifyRejectsModifiedMessage(t *testing.T) {
	provider := NewECDSASHA256Provider()
	privateKeyPEM, publicKeyPEM := generateECDSAKeyPairPEM(t)

	signature, err := provider.Sign([]byte("original message"), privateKeyPEM)
	if err != nil {
		t.Fatalf("sign message: %v", err)
	}

	if err := provider.Verify([]byte("modified message"), signature, publicKeyPEM); err == nil {
		t.Fatal("expected modified message verification to fail")
	}
}

func TestECDSASHA256ProviderVerifyRejectsWrongPublicKey(t *testing.T) {
	provider := NewECDSASHA256Provider()
	privateKeyPEM, _ := generateECDSAKeyPairPEM(t)
	_, wrongPublicKeyPEM := generateECDSAKeyPairPEM(t)
	message := []byte("message signed by another key")

	signature, err := provider.Sign(message, privateKeyPEM)
	if err != nil {
		t.Fatalf("sign message: %v", err)
	}

	if err := provider.Verify(message, signature, wrongPublicKeyPEM); err == nil {
		t.Fatal("expected wrong public key verification to fail")
	}
}

func TestECDSASHA256ProviderVerifyRejectsCorruptedSignature(t *testing.T) {
	provider := NewECDSASHA256Provider()
	privateKeyPEM, publicKeyPEM := generateECDSAKeyPairPEM(t)
	message := []byte("message with corrupted signature")

	signature, err := provider.Sign(message, privateKeyPEM)
	if err != nil {
		t.Fatalf("sign message: %v", err)
	}
	signature[0] ^= 0xff

	if err := provider.Verify(message, signature, publicKeyPEM); err == nil {
		t.Fatal("expected corrupted signature verification to fail")
	}
}

func generateECDSAKeyPairPEM(t *testing.T) ([]byte, []byte) {
	t.Helper()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate private key: %v", err)
	}

	privateKeyDER, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		t.Fatalf("marshal private key: %v", err)
	}

	publicKeyDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}

	return pem.EncodeToMemory(&pem.Block{
			Type:  "EC PRIVATE KEY",
			Bytes: privateKeyDER,
		}), pem.EncodeToMemory(&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: publicKeyDER,
		})
}
