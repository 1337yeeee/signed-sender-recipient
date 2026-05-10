package crypto

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
)

const ECDSASHA256Algorithm = "ECDSA-SHA256"

type ECDSASHA256Provider struct{}

func NewECDSASHA256Provider() *ECDSASHA256Provider {
	return &ECDSASHA256Provider{}
}

func (p *ECDSASHA256Provider) Hash(message []byte) []byte {
	sum := sha256.Sum256(message)
	return sum[:]
}

func (p *ECDSASHA256Provider) Verify(message []byte, signature []byte, publicKey []byte) error {
	parsedPublicKey, err := parseECDSAPublicKey(publicKey)
	if err != nil {
		return err
	}

	if !ecdsa.VerifyASN1(parsedPublicKey, p.Hash(message), signature) {
		return errors.New("invalid ECDSA-SHA256 signature")
	}

	return nil
}

func (p *ECDSASHA256Provider) Sign(message []byte, privateKey []byte) ([]byte, error) {
	parsedPrivateKey, err := parseECDSAPrivateKey(privateKey)
	if err != nil {
		return nil, err
	}

	return ecdsa.SignASN1(rand.Reader, parsedPrivateKey, p.Hash(message))
}

func parseECDSAPrivateKey(key []byte) (*ecdsa.PrivateKey, error) {
	der := decodePEM(key)

	if privateKey, err := x509.ParseECPrivateKey(der); err == nil {
		return privateKey, nil
	}

	parsedKey, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		return nil, fmt.Errorf("parse ECDSA private key: %w", err)
	}

	privateKey, ok := parsedKey.(*ecdsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key is not ECDSA")
	}

	return privateKey, nil
}

func parseECDSAPublicKey(key []byte) (*ecdsa.PublicKey, error) {
	parsedKey, err := x509.ParsePKIXPublicKey(decodePEM(key))
	if err != nil {
		return nil, fmt.Errorf("parse ECDSA public key: %w", err)
	}

	publicKey, ok := parsedKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("public key is not ECDSA")
	}

	return publicKey, nil
}

func decodePEM(data []byte) []byte {
	block, _ := pem.Decode(data)
	if block == nil {
		return data
	}

	return block.Bytes
}
