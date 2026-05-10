package usecase

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"electronic-digital-signature/internal/infra/encryption"
)

var (
	ErrInvalidEncryptedPackage = errors.New("invalid encrypted package")
	ErrInvalidSignature        = errors.New("invalid signature")
)

type decryptPackageVerifier interface {
	Hash(message []byte) []byte
	Verify(message []byte, signature []byte, publicKey []byte) error
}

type packageDecryptor interface {
	Decrypt(pkg encryption.EncryptedPackage) ([]byte, error)
}

type VerifyDecryptPackageInput struct {
	PackageContent []byte
}

type VerifyDecryptPackageMetadata struct {
	DocumentID           string
	Version              string
	SenderEmail          string
	SenderKeyFingerprint string
	SignedAt             string
	EncryptionAlgorithm  string
	KeyTransport         string
	SignatureAlgorithm   string
	OriginalFileName     string
	MimeType             string
	HashBase64           string
}

type VerifyDecryptPackageResult struct {
	Metadata          VerifyDecryptPackageMetadata
	DecryptedDocument []byte
}

type VerifyDecryptPackageUseCase struct {
	decryptor packageDecryptor
	verifier  decryptPackageVerifier
}

func NewVerifyDecryptPackageUseCase(
	decryptor packageDecryptor,
	verifier decryptPackageVerifier,
) *VerifyDecryptPackageUseCase {
	return &VerifyDecryptPackageUseCase{
		decryptor: decryptor,
		verifier:  verifier,
	}
}

func (uc *VerifyDecryptPackageUseCase) Execute(ctx context.Context, input VerifyDecryptPackageInput) (*VerifyDecryptPackageResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if uc.decryptor == nil {
		return nil, fmt.Errorf("package decryptor is not configured")
	}
	if uc.verifier == nil {
		return nil, fmt.Errorf("signature verifier is not configured")
	}
	if len(input.PackageContent) == 0 {
		return nil, fmt.Errorf("encrypted package is required")
	}

	pkg, err := encryption.DecodePackage(input.PackageContent)
	if err != nil {
		return nil, fmt.Errorf("%w", ErrInvalidEncryptedPackage)
	}
	if strings.TrimSpace(pkg.SenderPublicKeyPEM) == "" {
		return nil, fmt.Errorf("%w", ErrInvalidEncryptedPackage)
	}

	decryptedDocument, err := uc.decryptor.Decrypt(pkg)
	if err != nil {
		return nil, fmt.Errorf("%w", ErrInvalidEncryptedPackage)
	}
	if encodedHash := base64.StdEncoding.EncodeToString(uc.verifier.Hash(decryptedDocument)); encodedHash != pkg.HashBase64 {
		return nil, fmt.Errorf("%w", ErrInvalidEncryptedPackage)
	}

	signature, err := base64.StdEncoding.DecodeString(pkg.SignatureBase64)
	if err != nil {
		return nil, fmt.Errorf("%w", ErrInvalidEncryptedPackage)
	}
	if err := uc.verifier.Verify(decryptedDocument, signature, []byte(pkg.SenderPublicKeyPEM)); err != nil {
		return nil, fmt.Errorf("%w", ErrInvalidSignature)
	}

	return &VerifyDecryptPackageResult{
		Metadata:          metadataFromPackage(pkg),
		DecryptedDocument: decryptedDocument,
	}, nil
}

func metadataFromPackage(pkg encryption.EncryptedPackage) VerifyDecryptPackageMetadata {
	return VerifyDecryptPackageMetadata{
		DocumentID:           pkg.DocumentID,
		Version:              pkg.Version,
		SenderEmail:          pkg.SenderEmail,
		SenderKeyFingerprint: pkg.SenderPublicKeyFingerprint,
		SignedAt:             pkg.SignedAt,
		EncryptionAlgorithm:  pkg.EncryptionAlgorithm,
		KeyTransport:         pkg.KeyTransport,
		SignatureAlgorithm:   pkg.SignatureAlgorithm,
		OriginalFileName:     pkg.OriginalFileName,
		MimeType:             pkg.MimeType,
		HashBase64:           pkg.HashBase64,
	}
}
