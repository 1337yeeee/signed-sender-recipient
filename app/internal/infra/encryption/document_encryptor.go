package encryption

import (
	"context"
	"fmt"

	"electronic-digital-signature/internal/domain/model"
)

type encryptedPackageStorage interface {
	SaveEncryptedPackage(ctx context.Context, documentID string, content []byte) (string, error)
}

type DocumentEncryptor struct {
	aesGCMEncryptor *AESGCMEncryptor
	storage         encryptedPackageStorage
}

func NewDocumentEncryptor(storage encryptedPackageStorage) *DocumentEncryptor {
	return &DocumentEncryptor{
		aesGCMEncryptor: NewAESGCMEncryptor(),
		storage:         storage,
	}
}

func (e *DocumentEncryptor) EncryptAndSave(ctx context.Context, document model.Document, content []byte) (EncryptedPackage, string, error) {
	if e.storage == nil {
		return EncryptedPackage{}, "", fmt.Errorf("encrypted package storage is not configured")
	}

	pkg, err := e.aesGCMEncryptor.EncryptDocument(document, content)
	if err != nil {
		return EncryptedPackage{}, "", err
	}

	encodedPackage, err := EncodePackage(pkg)
	if err != nil {
		return EncryptedPackage{}, "", err
	}

	path, err := e.storage.SaveEncryptedPackage(ctx, document.ID, encodedPackage)
	if err != nil {
		return EncryptedPackage{}, "", err
	}

	return pkg, path, nil
}
