package model

import "time"

type Document struct {
	ID                         string     `json:"id"`
	SenderEmail                string     `json:"sender_email"`
	RecipientEmail             string     `json:"recipient_email"`
	OriginalFileName           string     `json:"original_file_name"`
	StoredPath                 string     `json:"stored_path"`
	MimeType                   string     `json:"mime_type"`
	Hash                       []byte     `json:"-"`
	Signature                  []byte     `json:"-"`
	SenderPublicKeyPEM         string     `json:"sender_public_key_pem"`
	SenderPublicKeyFingerprint string     `json:"sender_public_key_fingerprint"`
	EncryptedPath              string     `json:"encrypted_path"`
	CreatedAt                  time.Time  `json:"created_at"`
	SignedAt                   time.Time  `json:"signed_at"`
	SentAt                     *time.Time `json:"sent_at,omitempty"`
}

type VisualMetadata struct {
	DocumentID        string
	SenderEmail       string
	SenderFingerprint string
	SignedAt          time.Time
	Version           string
}
