package encryption

import (
	"encoding/json"
	"fmt"
)

type EncryptedPackage struct {
	Version                    string `json:"version"`
	DocumentID                 string `json:"document_id"`
	SenderEmail                string `json:"sender_email"`
	SenderPublicKeyPEM         string `json:"sender_public_key_pem"`
	SenderPublicKeyFingerprint string `json:"sender_public_key_fingerprint"`
	SignedAt                   string `json:"signed_at"`
	EncryptionAlgorithm        string `json:"encryption_algorithm"`
	KeyTransport               string `json:"key_transport"`
	EncryptedKeyBase64         string `json:"encrypted_key_base64"`
	NonceBase64                string `json:"nonce_base64"`
	CiphertextBase64           string `json:"ciphertext_base64"`
	SignatureBase64            string `json:"signature_base64"`
	HashBase64                 string `json:"hash_base64"`
	SignatureAlgorithm         string `json:"signature_algorithm"`
	OriginalFileName           string `json:"original_file_name"`
	MimeType                   string `json:"mime_type"`
}

func EncodePackage(pkg EncryptedPackage) ([]byte, error) {
	encoded, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode encrypted package: %w", err)
	}

	return encoded, nil
}

func DecodePackage(content []byte) (EncryptedPackage, error) {
	var pkg EncryptedPackage
	if err := json.Unmarshal(content, &pkg); err != nil {
		return EncryptedPackage{}, fmt.Errorf("decode encrypted package: %w", err)
	}

	return pkg, nil
}
