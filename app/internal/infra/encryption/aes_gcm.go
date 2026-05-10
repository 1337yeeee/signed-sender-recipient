package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"time"

	"electronic-digital-signature/internal/domain/model"
	"electronic-digital-signature/internal/infra/crypto"
)

const (
	PackageVersion     = "2"
	AESGCMAlgorithm    = "AES-256-GCM"
	PlaintextDemoKey   = "plaintext_demo"
	SignatureAlgorithm = crypto.ECDSASHA256Algorithm
	aes256KeySize      = 32
	gcmNonceSize       = 12
)

type AESGCMEncryptor struct{}

func NewAESGCMEncryptor() *AESGCMEncryptor {
	return &AESGCMEncryptor{}
}

func (e *AESGCMEncryptor) EncryptDocument(document model.Document, content []byte) (EncryptedPackage, error) {
	key, err := randomBytes(aes256KeySize)
	if err != nil {
		return EncryptedPackage{}, err
	}

	nonce, ciphertext, err := encryptAESGCM(key, content)
	if err != nil {
		return EncryptedPackage{}, err
	}

	return EncryptedPackage{
		Version:                    PackageVersion,
		DocumentID:                 document.ID,
		SenderEmail:                document.SenderEmail,
		SenderPublicKeyPEM:         document.SenderPublicKeyPEM,
		SenderPublicKeyFingerprint: document.SenderPublicKeyFingerprint,
		SignedAt:                   document.SignedAt.UTC().Format(time.RFC3339Nano),
		EncryptionAlgorithm:        AESGCMAlgorithm,
		KeyTransport:               PlaintextDemoKey,
		EncryptedKeyBase64:         base64.StdEncoding.EncodeToString(key),
		NonceBase64:                base64.StdEncoding.EncodeToString(nonce),
		CiphertextBase64:           base64.StdEncoding.EncodeToString(ciphertext),
		SignatureBase64:            base64.StdEncoding.EncodeToString(document.Signature),
		HashBase64:                 base64.StdEncoding.EncodeToString(document.Hash),
		SignatureAlgorithm:         SignatureAlgorithm,
		OriginalFileName:           document.OriginalFileName,
		MimeType:                   document.MimeType,
	}, nil
}

func (e *AESGCMEncryptor) Decrypt(pkg EncryptedPackage) ([]byte, error) {
	if pkg.EncryptionAlgorithm != AESGCMAlgorithm {
		return nil, fmt.Errorf("unsupported encryption algorithm: %s", pkg.EncryptionAlgorithm)
	}
	if pkg.KeyTransport != PlaintextDemoKey {
		return nil, fmt.Errorf("unsupported key transport: %s", pkg.KeyTransport)
	}

	key, err := base64.StdEncoding.DecodeString(pkg.EncryptedKeyBase64)
	if err != nil {
		return nil, fmt.Errorf("decode encrypted key: %w", err)
	}
	nonce, err := base64.StdEncoding.DecodeString(pkg.NonceBase64)
	if err != nil {
		return nil, fmt.Errorf("decode nonce: %w", err)
	}
	ciphertext, err := base64.StdEncoding.DecodeString(pkg.CiphertextBase64)
	if err != nil {
		return nil, fmt.Errorf("decode ciphertext: %w", err)
	}

	return decryptAESGCM(key, nonce, ciphertext)
}

func encryptAESGCM(key []byte, content []byte) ([]byte, []byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, fmt.Errorf("create AES cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("create GCM: %w", err)
	}

	nonce, err := randomBytes(gcmNonceSize)
	if err != nil {
		return nil, nil, err
	}

	return nonce, aead.Seal(nil, nonce, content, nil), nil
}

func decryptAESGCM(key []byte, nonce []byte, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	content, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt AES-GCM content: %w", err)
	}

	return content, nil
}

func randomBytes(size int) ([]byte, error) {
	bytes := make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		return nil, fmt.Errorf("generate random bytes: %w", err)
	}

	return bytes, nil
}
