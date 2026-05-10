package model

import "time"

type PackageStatus string

const (
	PackageStatusReceived   PackageStatus = "received"
	PackageStatusProcessing PackageStatus = "processing"
	PackageStatusProcessed  PackageStatus = "processed"
	PackageStatusFailed     PackageStatus = "failed"
)

type MailQuery struct {
	Limit int
}

type MailAddress struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type MailAttachment struct {
	ID          string `json:"id"`
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size,omitempty"`
}

type MailMessage struct {
	ID          string           `json:"id"`
	Subject     string           `json:"subject"`
	From        MailAddress      `json:"from"`
	To          []MailAddress    `json:"to"`
	CreatedAt   time.Time        `json:"created_at"`
	Attachments []MailAttachment `json:"attachments"`
}

type InboundPackage struct {
	MailMessageID         string        `json:"mail_message_id"`
	RecipientEmail        string        `json:"recipient_email"`
	Subject               string        `json:"subject"`
	MailReceivedAt        time.Time     `json:"mail_received_at"`
	ProcessedAt           time.Time     `json:"processed_at"`
	Status                PackageStatus `json:"status"`
	PackageFileName       string        `json:"package_file_name"`
	PackagePath           string        `json:"package_path"`
	DecryptedDocumentPath string        `json:"decrypted_document_path,omitempty"`
	SignatureValid        bool          `json:"signature_valid"`
	ErrorMessage          string        `json:"error_message,omitempty"`

	DocumentID           string `json:"document_id,omitempty"`
	SenderEmail          string `json:"sender_email,omitempty"`
	SenderKeyFingerprint string `json:"sender_public_key_fingerprint,omitempty"`
	SignedAt             string `json:"signed_at,omitempty"`
	EncryptionAlgorithm  string `json:"encryption_algorithm,omitempty"`
	KeyTransport         string `json:"key_transport,omitempty"`
	SignatureAlgorithm   string `json:"signature_algorithm,omitempty"`
	OriginalFileName     string `json:"original_file_name,omitempty"`
	MimeType             string `json:"mime_type,omitempty"`
	HashBase64           string `json:"hash_base64,omitempty"`
}
