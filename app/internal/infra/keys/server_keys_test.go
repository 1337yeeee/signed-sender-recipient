package keys

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOrCreateKeyPairCreatesMissingFiles(t *testing.T) {
	dir := t.TempDir()
	privateKeyPath := filepath.Join(dir, "private.pem")
	publicKeyPath := filepath.Join(dir, "public.pem")

	keyPair, err := LoadOrCreateKeyPair(privateKeyPath, publicKeyPath, "", "")
	if err != nil {
		t.Fatalf("load or create key pair: %v", err)
	}

	if len(keyPair.PrivateKey) == 0 {
		t.Fatal("expected private key bytes")
	}
	if len(keyPair.PublicKey) == 0 {
		t.Fatal("expected public key bytes")
	}
	if _, err := os.Stat(privateKeyPath); err != nil {
		t.Fatalf("expected private key file, got %v", err)
	}
	if _, err := os.Stat(publicKeyPath); err != nil {
		t.Fatalf("expected public key file, got %v", err)
	}
}

func TestLoadOrCreateKeyPairLoadsExistingFiles(t *testing.T) {
	dir := t.TempDir()
	privateKeyPath := filepath.Join(dir, "private.pem")
	publicKeyPath := filepath.Join(dir, "public.pem")

	initial, err := LoadOrCreateKeyPair(privateKeyPath, publicKeyPath, "", "")
	if err != nil {
		t.Fatalf("create initial key pair: %v", err)
	}

	loaded, err := LoadOrCreateKeyPair(privateKeyPath, publicKeyPath, "", "")
	if err != nil {
		t.Fatalf("reload key pair: %v", err)
	}

	if string(initial.PrivateKey) != string(loaded.PrivateKey) {
		t.Fatal("expected private key to be loaded from disk")
	}
	if string(initial.PublicKey) != string(loaded.PublicKey) {
		t.Fatal("expected public key to be loaded from disk")
	}
}

func TestLoadOrCreateKeyPairRejectsIncompletePair(t *testing.T) {
	dir := t.TempDir()
	privateKeyPath := filepath.Join(dir, "private.pem")
	publicKeyPath := filepath.Join(dir, "public.pem")

	if err := os.WriteFile(privateKeyPath, []byte("private key"), 0o600); err != nil {
		t.Fatalf("write private key: %v", err)
	}

	if _, err := LoadOrCreateKeyPair(privateKeyPath, publicKeyPath, "", ""); err == nil {
		t.Fatal("expected incomplete key pair to fail")
	}
}

func TestPublicKeyFingerprint(t *testing.T) {
	dir := t.TempDir()
	privateKeyPath := filepath.Join(dir, "private.pem")
	publicKeyPath := filepath.Join(dir, "public.pem")

	keyPair, err := LoadOrCreateKeyPair(privateKeyPath, publicKeyPath, "", "")
	if err != nil {
		t.Fatalf("create key pair: %v", err)
	}

	fingerprint, err := PublicKeyFingerprint(keyPair.PublicKey)
	if err != nil {
		t.Fatalf("calculate fingerprint: %v", err)
	}
	if fingerprint == "" {
		t.Fatal("expected non-empty fingerprint")
	}
}
