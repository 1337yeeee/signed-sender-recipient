package encryption

import (
	"context"
	"encoding/base64"
	"testing"

	"electronic-digital-signature/internal/domain/model"
)

func TestAESGCMEncryptorEncryptsAndDecryptsDocument(t *testing.T) {
	encryptor := NewAESGCMEncryptor()
	document := model.Document{
		ID:                         "document-id",
		SenderEmail:                "sender@example.com",
		SenderPublicKeyPEM:         "public-key",
		SenderPublicKeyFingerprint: "fingerprint",
		OriginalFileName:           "contract.docx",
		MimeType:                   "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		Hash:                       []byte("document hash"),
		Signature:                  []byte("document signature"),
	}
	content := []byte("final docx bytes")

	pkg, err := encryptor.EncryptDocument(document, content)
	if err != nil {
		t.Fatalf("encrypt document: %v", err)
	}

	if pkg.Version != PackageVersion {
		t.Fatalf("expected package version %q, got %q", PackageVersion, pkg.Version)
	}
	if pkg.EncryptionAlgorithm != AESGCMAlgorithm {
		t.Fatalf("expected encryption algorithm %q, got %q", AESGCMAlgorithm, pkg.EncryptionAlgorithm)
	}
	if pkg.KeyTransport != PlaintextDemoKey {
		t.Fatalf("expected key transport %q, got %q", PlaintextDemoKey, pkg.KeyTransport)
	}
	if pkg.CiphertextBase64 == base64.StdEncoding.EncodeToString(content) {
		t.Fatal("expected ciphertext to differ from plaintext")
	}

	decrypted, err := encryptor.Decrypt(pkg)
	if err != nil {
		t.Fatalf("decrypt document package: %v", err)
	}
	if string(decrypted) != string(content) {
		t.Fatalf("expected decrypted content %q, got %q", content, decrypted)
	}
}

func TestAESGCMEncryptorRejectsTamperedCiphertext(t *testing.T) {
	encryptor := NewAESGCMEncryptor()
	pkg, err := encryptor.EncryptDocument(model.Document{ID: "document-id"}, []byte("final docx bytes"))
	if err != nil {
		t.Fatalf("encrypt document: %v", err)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(pkg.CiphertextBase64)
	if err != nil {
		t.Fatalf("decode ciphertext: %v", err)
	}
	ciphertext[0] ^= 0xff
	pkg.CiphertextBase64 = base64.StdEncoding.EncodeToString(ciphertext)

	if _, err := encryptor.Decrypt(pkg); err == nil {
		t.Fatal("expected tampered ciphertext decrypt to fail")
	}
}

func TestEncryptedPackageEncodeDecode(t *testing.T) {
	original := EncryptedPackage{
		Version:                    PackageVersion,
		DocumentID:                 "document-id",
		SenderEmail:                "sender@example.com",
		SenderPublicKeyPEM:         "public-key",
		SenderPublicKeyFingerprint: "fingerprint",
		SignedAt:                   "2026-05-10T10:00:00Z",
		EncryptionAlgorithm:        AESGCMAlgorithm,
		KeyTransport:               PlaintextDemoKey,
		EncryptedKeyBase64:         "key",
		NonceBase64:                "nonce",
		CiphertextBase64:           "ciphertext",
		SignatureBase64:            "signature",
		HashBase64:                 "hash",
		SignatureAlgorithm:         SignatureAlgorithm,
		OriginalFileName:           "contract.docx",
		MimeType:                   "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	}

	encoded, err := EncodePackage(original)
	if err != nil {
		t.Fatalf("encode package: %v", err)
	}

	decoded, err := DecodePackage(encoded)
	if err != nil {
		t.Fatalf("decode package: %v", err)
	}

	if decoded != original {
		t.Fatalf("expected decoded package %+v, got %+v", original, decoded)
	}
}

func TestDocumentEncryptorEncryptAndSave(t *testing.T) {
	storage := &fakeEncryptedPackageStorage{}
	encryptor := NewDocumentEncryptor(storage)
	document := model.Document{
		ID:                         "document-id",
		SenderEmail:                "sender@example.com",
		SenderPublicKeyPEM:         "public-key",
		SenderPublicKeyFingerprint: "fingerprint",
		OriginalFileName:           "contract.docx",
		MimeType:                   "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		Hash:                       []byte("document hash"),
		Signature:                  []byte("document signature"),
	}
	content := []byte("final docx bytes")

	pkg, path, err := encryptor.EncryptAndSave(context.Background(), document, content)
	if err != nil {
		t.Fatalf("encrypt and save document: %v", err)
	}

	if path != "stored/document-id.json" {
		t.Fatalf("expected package path, got %q", path)
	}
	if storage.documentID != document.ID {
		t.Fatalf("expected stored document id %q, got %q", document.ID, storage.documentID)
	}

	storedPackage, err := DecodePackage(storage.content)
	if err != nil {
		t.Fatalf("decode stored package: %v", err)
	}
	if storedPackage.DocumentID != document.ID {
		t.Fatalf("expected stored package document id %q, got %q", document.ID, storedPackage.DocumentID)
	}

	decrypted, err := NewAESGCMEncryptor().Decrypt(pkg)
	if err != nil {
		t.Fatalf("decrypt returned package: %v", err)
	}
	if string(decrypted) != string(content) {
		t.Fatalf("expected decrypted content %q, got %q", content, decrypted)
	}
}

type fakeEncryptedPackageStorage struct {
	documentID string
	content    []byte
}

func (s *fakeEncryptedPackageStorage) SaveEncryptedPackage(_ context.Context, documentID string, content []byte) (string, error) {
	s.documentID = documentID
	s.content = append([]byte(nil), content...)
	return "stored/" + documentID + ".json", nil
}
