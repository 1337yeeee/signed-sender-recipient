package dto

type InboundPackageResponse struct {
	MailMessageID              string `json:"mail_message_id"`
	RecipientEmail             string `json:"recipient_email"`
	Subject                    string `json:"subject"`
	MailReceivedAt             string `json:"mail_received_at"`
	ProcessedAt                string `json:"processed_at"`
	Status                     string `json:"status"`
	PackageFileName            string `json:"package_file_name"`
	PackagePath                string `json:"package_path"`
	DecryptedDocumentPath      string `json:"decrypted_document_path,omitempty"`
	SignatureValid             bool   `json:"signature_valid"`
	ErrorMessage               string `json:"error_message,omitempty"`
	DocumentID                 string `json:"document_id,omitempty"`
	SenderEmail                string `json:"sender_email,omitempty"`
	SenderPublicKeyFingerprint string `json:"sender_public_key_fingerprint,omitempty"`
	SignedAt                   string `json:"signed_at,omitempty"`
	EncryptionAlgorithm        string `json:"encryption_algorithm,omitempty"`
	KeyTransport               string `json:"key_transport,omitempty"`
	SignatureAlgorithm         string `json:"signature_algorithm,omitempty"`
	OriginalFileName           string `json:"original_file_name,omitempty"`
	MimeType                   string `json:"mime_type,omitempty"`
	HashBase64                 string `json:"hash_base64,omitempty"`
}

type InboundPackageEventResponse struct {
	Type      string                 `json:"type"`
	Timestamp string                 `json:"timestamp"`
	Package   InboundPackageResponse `json:"package"`
}
