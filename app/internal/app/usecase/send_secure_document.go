package usecase

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"electronic-digital-signature/internal/domain/model"
	"electronic-digital-signature/internal/infra/encryption"
)

const MaxUploadDocumentSizeBytes = 10 << 20

var AllowedUploadDocumentMIMETypes = map[string]struct{}{
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": {},
	"application/octet-stream": {},
}

type EmailAttachment struct {
	FileName    string
	ContentType string
	Content     []byte
}

type Mailer interface {
	SendEmail(ctx context.Context, to []string, subject, body string, attachments []EmailAttachment) error
}

type sendDocumentStorage interface {
	Save(ctx context.Context, id, originalFileName string, content io.Reader) (string, error)
	SaveEncryptedPackage(ctx context.Context, documentID string, content []byte) (string, error)
	Read(ctx context.Context, path string) ([]byte, error)
}

type documentIDGenerator interface {
	Generate() (string, error)
}

type documentProcessor interface {
	AddMetadata(content []byte, metadata model.VisualMetadata) ([]byte, error)
}

type DocumentSigner interface {
	Hash(message []byte) []byte
	Sign(message []byte, privateKey []byte) ([]byte, error)
}

type DocumentEncryptor interface {
	EncryptAndSave(ctx context.Context, document model.Document, content []byte) (encryption.EncryptedPackage, string, error)
}

type SendSecureDocumentInput struct {
	Document         model.Document
	To               []string
	Subject          string
	EncryptedPackage []byte
	AttachmentName   string
}

type SendDocumentInput struct {
	RecipientEmail   string
	OriginalFileName string
	MimeType         string
	Content          io.Reader
}

type SendDocumentResult struct {
	DocumentID       string
	SenderEmail      string
	RecipientEmail   string
	OriginalFileName string
	StoredPath       string
	PackagePath      string
	MimeType         string
	CreatedAt        time.Time
	SignedAt         time.Time
	SentAt           time.Time
}

type SendDocumentUseCase struct {
	storage           sendDocumentStorage
	idGenerator       documentIDGenerator
	processor         documentProcessor
	signer            DocumentSigner
	privateKey        []byte
	publicKey         []byte
	senderEmail       string
	senderFingerprint string
	encryptor         DocumentEncryptor
	mailer            Mailer
}

func NewSendDocumentUseCase(
	storage sendDocumentStorage,
	idGenerator documentIDGenerator,
	processor documentProcessor,
	signer DocumentSigner,
	privateKey []byte,
	publicKey []byte,
	senderEmail string,
	senderFingerprint string,
	encryptor DocumentEncryptor,
	mailer Mailer,
) *SendDocumentUseCase {
	return &SendDocumentUseCase{
		storage:           storage,
		idGenerator:       idGenerator,
		processor:         processor,
		signer:            signer,
		privateKey:        privateKey,
		publicKey:         publicKey,
		senderEmail:       senderEmail,
		senderFingerprint: senderFingerprint,
		encryptor:         encryptor,
		mailer:            mailer,
	}
}

func (uc *SendDocumentUseCase) Execute(ctx context.Context, input SendDocumentInput) (*SendDocumentResult, error) {
	if uc.storage == nil {
		return nil, fmt.Errorf("document storage is not configured")
	}
	if uc.idGenerator == nil {
		return nil, fmt.Errorf("document id generator is not configured")
	}
	if uc.processor == nil {
		return nil, fmt.Errorf("document processor is not configured")
	}
	if uc.signer == nil {
		return nil, fmt.Errorf("document signer is not configured")
	}
	if uc.encryptor == nil {
		return nil, fmt.Errorf("document encryptor is not configured")
	}
	if uc.mailer == nil {
		return nil, fmt.Errorf("mailer is not configured")
	}
	if len(uc.privateKey) == 0 {
		return nil, fmt.Errorf("private key is not configured")
	}
	if len(uc.publicKey) == 0 {
		return nil, fmt.Errorf("public key is not configured")
	}
	if strings.TrimSpace(uc.senderEmail) == "" {
		return nil, fmt.Errorf("sender email is required")
	}
	if strings.TrimSpace(input.RecipientEmail) == "" {
		return nil, fmt.Errorf("recipient email is required")
	}
	if input.Content == nil {
		return nil, fmt.Errorf("document file is required")
	}
	if !strings.EqualFold(filepath.Ext(input.OriginalFileName), ".docx") {
		return nil, fmt.Errorf("document file must have .docx extension")
	}

	documentID, err := uc.idGenerator.Generate()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	content, err := io.ReadAll(input.Content)
	if err != nil {
		return nil, fmt.Errorf("read uploaded document: %w", err)
	}

	updatedContent, err := uc.processor.AddMetadata(content, model.VisualMetadata{
		DocumentID:        documentID,
		SenderEmail:       uc.senderEmail,
		SenderFingerprint: uc.senderFingerprint,
		SignedAt:          now,
		Version:           encryption.PackageVersion,
	})
	if err != nil {
		return nil, err
	}

	documentHash := uc.signer.Hash(updatedContent)
	signature, err := uc.signer.Sign(updatedContent, uc.privateKey)
	if err != nil {
		return nil, fmt.Errorf("sign document: %w", err)
	}

	storedPath, err := uc.storage.Save(ctx, documentID, input.OriginalFileName, bytes.NewReader(updatedContent))
	if err != nil {
		return nil, fmt.Errorf("save signed document: %w", err)
	}

	document := model.Document{
		ID:                         documentID,
		SenderEmail:                uc.senderEmail,
		RecipientEmail:             strings.TrimSpace(input.RecipientEmail),
		OriginalFileName:           input.OriginalFileName,
		StoredPath:                 storedPath,
		MimeType:                   input.MimeType,
		Hash:                       documentHash,
		Signature:                  signature,
		SenderPublicKeyPEM:         string(uc.publicKey),
		SenderPublicKeyFingerprint: uc.senderFingerprint,
		CreatedAt:                  now,
		SignedAt:                   now,
	}

	_, packagePath, err := uc.encryptor.EncryptAndSave(ctx, document, updatedContent)
	if err != nil {
		return nil, fmt.Errorf("create encrypted package: %w", err)
	}

	encryptedPackage, err := uc.storage.Read(ctx, packagePath)
	if err != nil {
		return nil, fmt.Errorf("read encrypted package: %w", err)
	}

	if err := SendSecureDocument(ctx, uc.mailer, SendSecureDocumentInput{
		Document:         document,
		To:               []string{document.RecipientEmail},
		Subject:          fmt.Sprintf("Encrypted document package: %s", document.OriginalFileName),
		EncryptedPackage: encryptedPackage,
		AttachmentName:   filepath.Base(packagePath),
	}); err != nil {
		return nil, fmt.Errorf("send email: %w", err)
	}

	sentAt := time.Now().UTC()
	document.EncryptedPath = packagePath
	document.SentAt = &sentAt

	return &SendDocumentResult{
		DocumentID:       document.ID,
		SenderEmail:      document.SenderEmail,
		RecipientEmail:   document.RecipientEmail,
		OriginalFileName: document.OriginalFileName,
		StoredPath:       document.StoredPath,
		PackagePath:      document.EncryptedPath,
		MimeType:         document.MimeType,
		CreatedAt:        document.CreatedAt,
		SignedAt:         document.SignedAt,
		SentAt:           sentAt,
	}, nil
}

func SendSecureDocument(ctx context.Context, mailer Mailer, input SendSecureDocumentInput) error {
	if len(input.EncryptedPackage) == 0 {
		return fmt.Errorf("encrypted package is required")
	}

	attachmentName := input.AttachmentName
	if attachmentName == "" {
		attachmentName = input.Document.ID + "_encrypted_package.json"
	}

	body := fmt.Sprintf(
		"Encrypted document package is attached.\n\nDocument ID: %s\nSender: %s\nEncryption algorithm: %s\nKey transport: %s\nSignature algorithm: %s\n\nThe package contains the sender public key and signature metadata required for autonomous verification after decryption.",
		input.Document.ID,
		input.Document.SenderEmail,
		encryption.AESGCMAlgorithm,
		encryption.PlaintextDemoKey,
		encryption.SignatureAlgorithm,
	)

	return mailer.SendEmail(ctx, input.To, input.Subject, body, []EmailAttachment{
		{
			FileName:    attachmentName,
			ContentType: "application/json",
			Content:     input.EncryptedPackage,
		},
	})
}
