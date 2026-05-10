package keys

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type KeyPair struct {
	PrivateKey []byte
	PublicKey  []byte
}

func LoadOrCreateKeyPair(privateKeyPath, publicKeyPath, privateKeyPEM, publicKeyPEM string) (KeyPair, error) {
	if strings.TrimSpace(privateKeyPEM) != "" || strings.TrimSpace(publicKeyPEM) != "" {
		if strings.TrimSpace(privateKeyPEM) == "" || strings.TrimSpace(publicKeyPEM) == "" {
			return KeyPair{}, fmt.Errorf("both private and public key PEM values are required when using raw environment keys")
		}

		privateKey, err := validateRequiredKeyBytes("private key", []byte(privateKeyPEM), "environment")
		if err != nil {
			return KeyPair{}, err
		}
		publicKey, err := validateRequiredKeyBytes("public key", []byte(publicKeyPEM), "environment")
		if err != nil {
			return KeyPair{}, err
		}

		return KeyPair{
			PrivateKey: privateKey,
			PublicKey:  publicKey,
		}, nil
	}

	if strings.TrimSpace(privateKeyPath) == "" || strings.TrimSpace(publicKeyPath) == "" {
		return KeyPair{}, fmt.Errorf("private and public key paths are required")
	}

	privateExists := fileExists(privateKeyPath)
	publicExists := fileExists(publicKeyPath)

	switch {
	case privateExists && publicExists:
		privateKey, err := readRequiredKeyFile("private key", privateKeyPath)
		if err != nil {
			return KeyPair{}, err
		}
		publicKey, err := readRequiredKeyFile("public key", publicKeyPath)
		if err != nil {
			return KeyPair{}, err
		}

		return KeyPair{
			PrivateKey: privateKey,
			PublicKey:  publicKey,
		}, nil
	case privateExists != publicExists:
		return KeyPair{}, fmt.Errorf("key pair is incomplete: both %q and %q must exist", privateKeyPath, publicKeyPath)
	default:
		return generateAndSaveKeyPair(privateKeyPath, publicKeyPath)
	}
}

func PublicKeyFingerprint(publicKey []byte) (string, error) {
	block, _ := pem.Decode(publicKey)
	if block == nil {
		return "", fmt.Errorf("decode public key PEM")
	}

	parsedKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("parse public key: %w", err)
	}

	der, err := x509.MarshalPKIXPublicKey(parsedKey)
	if err != nil {
		return "", fmt.Errorf("marshal public key: %w", err)
	}

	sum := sha256.Sum256(der)
	return hex.EncodeToString(sum[:]), nil
}

func generateAndSaveKeyPair(privateKeyPath, publicKeyPath string) (KeyPair, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return KeyPair{}, fmt.Errorf("generate key pair: %w", err)
	}

	privateDER, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return KeyPair{}, fmt.Errorf("marshal private key: %w", err)
	}

	publicDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return KeyPair{}, fmt.Errorf("marshal public key: %w", err)
	}

	privatePEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: privateDER,
	})
	publicPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicDER,
	})

	if err := os.MkdirAll(filepath.Dir(privateKeyPath), 0o755); err != nil {
		return KeyPair{}, fmt.Errorf("create private key directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(publicKeyPath), 0o755); err != nil {
		return KeyPair{}, fmt.Errorf("create public key directory: %w", err)
	}
	if err := os.WriteFile(privateKeyPath, privatePEM, 0o600); err != nil {
		return KeyPair{}, fmt.Errorf("write private key: %w", err)
	}
	if err := os.WriteFile(publicKeyPath, publicPEM, 0o644); err != nil {
		return KeyPair{}, fmt.Errorf("write public key: %w", err)
	}

	return KeyPair{
		PrivateKey: privatePEM,
		PublicKey:  publicPEM,
	}, nil
}

func readRequiredKeyFile(name, path string) ([]byte, error) {
	key, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%s file does not exist: %s", name, path)
		}

		return nil, fmt.Errorf("read %s file %q: %w", name, path, err)
	}

	return validateRequiredKeyBytes(name, key, path)
}

func validateRequiredKeyBytes(name string, key []byte, source string) ([]byte, error) {
	if len(strings.TrimSpace(string(key))) == 0 {
		return nil, fmt.Errorf("%s file is empty: %s", name, source)
	}

	return key, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
