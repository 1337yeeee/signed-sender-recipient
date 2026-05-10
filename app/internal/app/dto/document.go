package dto

type SendDocumentResponse struct {
	DocumentID       string `json:"document_id"`
	SenderEmail      string `json:"sender_email"`
	RecipientEmail   string `json:"recipient_email"`
	OriginalFileName string `json:"original_file_name"`
	StoredPath       string `json:"stored_path"`
	PackagePath      string `json:"package_path"`
	MimeType         string `json:"mime_type"`
	CreatedAt        string `json:"created_at"`
	SignedAt         string `json:"signed_at"`
	SentAt           string `json:"sent_at"`
}

type VerifyDecryptPackageMetadata struct {
	DocumentID           string `json:"document_id"`
	Version              string `json:"version"`
	SenderEmail          string `json:"sender_email"`
	SenderKeyFingerprint string `json:"sender_public_key_fingerprint"`
	SignedAt             string `json:"signed_at"`
	EncryptionAlgorithm  string `json:"encryption_algorithm"`
	KeyTransport         string `json:"key_transport"`
	SignatureAlgorithm   string `json:"signature_algorithm"`
	OriginalFileName     string `json:"original_file_name"`
	MimeType             string `json:"mime_type"`
	HashBase64           string `json:"hash_base64"`
}

type VerifyDecryptPackageResponse struct {
	Valid                   bool                         `json:"valid"`
	Metadata                VerifyDecryptPackageMetadata `json:"metadata"`
	DecryptedDocumentBase64 string                       `json:"decrypted_document_base64,omitempty"`
}
