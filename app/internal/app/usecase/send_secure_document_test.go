package usecase

import (
	"context"
	"io"
	"strings"
	"testing"

	"electronic-digital-signature/internal/domain/model"
	"electronic-digital-signature/internal/infra/encryption"
)

func TestSendDocumentUseCaseSignsEncryptsAndSendsPackage(t *testing.T) {
	storage := &fakeSendDocumentStorage{}
	signer := &fakeDocumentSigner{}
	encryptor := &fakeDocumentEncryptor{storage: storage}
	mailer := &fakeMailer{}
	processor := &fakeDocumentProcessor{}
	idGenerator := &fakeDocumentIDGenerator{id: "document-id"}

	result, err := NewSendDocumentUseCase(
		storage,
		idGenerator,
		processor,
		signer,
		[]byte("sender-private-key"),
		[]byte("sender-public-key"),
		"sender@example.com",
		"fingerprint",
		encryptor,
		mailer,
	).Execute(context.Background(), SendDocumentInput{
		RecipientEmail:   "recipient@example.com",
		OriginalFileName: "contract.docx",
		MimeType:         "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		Content:          strings.NewReader("document bytes"),
	})
	if err != nil {
		t.Fatalf("send document: %v", err)
	}

	if result.DocumentID != "document-id" {
		t.Fatalf("expected document id, got %q", result.DocumentID)
	}
	if result.PackagePath != "stored/document-id_encrypted_package.json" {
		t.Fatalf("expected package path, got %q", result.PackagePath)
	}
	if result.SenderEmail != "sender@example.com" {
		t.Fatalf("expected sender email, got %q", result.SenderEmail)
	}
	if string(signer.signedMessage) != "document bytes|meta" {
		t.Fatalf("expected signer to receive document content, got %q", signer.signedMessage)
	}
	if string(encryptor.document.Signature) != "signature" {
		t.Fatalf("expected encryptor to receive signed document, got signature %q", encryptor.document.Signature)
	}
	if processor.metadata.DocumentID != "document-id" {
		t.Fatalf("expected metadata document id, got %q", processor.metadata.DocumentID)
	}
	if processor.metadata.SenderEmail != "sender@example.com" {
		t.Fatalf("expected metadata sender email, got %q", processor.metadata.SenderEmail)
	}
	if len(mailer.attachments) != 1 {
		t.Fatalf("expected package attachment, got %d", len(mailer.attachments))
	}
	attachmentContent := string(mailer.attachments[0].Content)
	if strings.Contains(attachmentContent, "sender-private-key") {
		t.Fatalf("private key leaked into attachment: %q", attachmentContent)
	}
	if strings.Contains(mailer.body, "sender-private-key") {
		t.Fatalf("private key leaked into email body: %q", mailer.body)
	}
}

func TestSendSecureDocumentSendsEncryptedPackageAttachment(t *testing.T) {
	mailer := &fakeMailer{}
	document := model.Document{
		ID:               "document-id",
		SenderEmail:      "sender@example.com",
		OriginalFileName: "contract.docx",
	}
	encryptedPackage := []byte(`{"document_id":"document-id"}`)

	err := SendSecureDocument(context.Background(), mailer, SendSecureDocumentInput{
		Document:         document,
		To:               []string{"recipient@example.com"},
		Subject:          "Encrypted document",
		EncryptedPackage: encryptedPackage,
	})
	if err != nil {
		t.Fatalf("send secure document: %v", err)
	}

	if len(mailer.attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(mailer.attachments))
	}
	attachment := mailer.attachments[0]
	if attachment.FileName != "document-id_encrypted_package.json" {
		t.Fatalf("expected default attachment name, got %q", attachment.FileName)
	}
	if attachment.ContentType != "application/json" {
		t.Fatalf("expected attachment content type application/json, got %q", attachment.ContentType)
	}
	if string(attachment.Content) != string(encryptedPackage) {
		t.Fatalf("expected attachment content %q, got %q", encryptedPackage, attachment.Content)
	}
	if mailer.body == "" {
		t.Fatal("expected email body")
	}
	if !strings.Contains(mailer.body, "Document ID: document-id") {
		t.Fatalf("expected document_id in body, got %q", mailer.body)
	}
	if !strings.Contains(mailer.body, "Sender: sender@example.com") {
		t.Fatalf("expected sender email in body, got %q", mailer.body)
	}
}

func TestSendSecureDocumentRejectsEmptyEncryptedPackage(t *testing.T) {
	err := SendSecureDocument(context.Background(), &fakeMailer{}, SendSecureDocumentInput{
		Document: model.Document{ID: "document-id"},
		To:       []string{"recipient@example.com"},
		Subject:  "Encrypted document",
	})
	if err == nil {
		t.Fatal("expected empty encrypted package to fail")
	}
}

func TestSendDocumentUseCaseRejectsMissingRecipient(t *testing.T) {
	_, err := NewSendDocumentUseCase(
		&fakeSendDocumentStorage{},
		&fakeDocumentIDGenerator{id: "document-id"},
		&fakeDocumentProcessor{},
		&fakeDocumentSigner{},
		[]byte("private"),
		[]byte("public"),
		"sender@example.com",
		"fingerprint",
		&fakeDocumentEncryptor{storage: &fakeSendDocumentStorage{}},
		&fakeMailer{},
	).Execute(context.Background(), SendDocumentInput{
		OriginalFileName: "contract.docx",
		MimeType:         "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		Content:          strings.NewReader("document bytes"),
	})
	if err == nil || !strings.Contains(err.Error(), "recipient email is required") {
		t.Fatalf("expected recipient email validation, got %v", err)
	}
}

type fakeMailer struct {
	to          []string
	subject     string
	body        string
	attachments []EmailAttachment
}

func (m *fakeMailer) SendEmail(_ context.Context, to []string, subject, body string, attachments []EmailAttachment) error {
	m.to = to
	m.subject = subject
	m.body = body
	m.attachments = attachments
	return nil
}

type fakeDocumentIDGenerator struct {
	id string
}

func (g *fakeDocumentIDGenerator) Generate() (string, error) {
	return g.id, nil
}

type fakeDocumentProcessor struct {
	metadata model.VisualMetadata
}

func (p *fakeDocumentProcessor) AddMetadata(content []byte, metadata model.VisualMetadata) ([]byte, error) {
	p.metadata = metadata
	return append(append([]byte(nil), content...), []byte("|meta")...), nil
}

type fakeSendDocumentStorage struct {
	contentByPath map[string][]byte
}

func (s *fakeSendDocumentStorage) Save(_ context.Context, id, originalFileName string, content io.Reader) (string, error) {
	if s.contentByPath == nil {
		s.contentByPath = map[string][]byte{}
	}
	bytes, err := io.ReadAll(content)
	if err != nil {
		return "", err
	}
	path := "stored/" + id + "_" + originalFileName
	s.contentByPath[path] = bytes
	return path, nil
}

func (s *fakeSendDocumentStorage) SaveEncryptedPackage(_ context.Context, documentID string, content []byte) (string, error) {
	if s.contentByPath == nil {
		s.contentByPath = map[string][]byte{}
	}
	path := "stored/" + documentID + "_encrypted_package.json"
	s.contentByPath[path] = append([]byte(nil), content...)
	return path, nil
}

func (s *fakeSendDocumentStorage) Read(_ context.Context, path string) ([]byte, error) {
	return append([]byte(nil), s.contentByPath[path]...), nil
}

type fakeDocumentSigner struct {
	signedMessage []byte
	privateKey    []byte
}

func (s *fakeDocumentSigner) Hash(message []byte) []byte {
	return []byte("hash")
}

func (s *fakeDocumentSigner) Sign(message []byte, privateKey []byte) ([]byte, error) {
	s.signedMessage = append([]byte(nil), message...)
	s.privateKey = append([]byte(nil), privateKey...)
	return []byte("signature"), nil
}

type fakeDocumentEncryptor struct {
	storage  *fakeSendDocumentStorage
	document model.Document
	content  []byte
}

func (e *fakeDocumentEncryptor) EncryptAndSave(_ context.Context, document model.Document, content []byte) (encryption.EncryptedPackage, string, error) {
	e.document = document
	e.content = append([]byte(nil), content...)
	path := "stored/" + document.ID + "_encrypted_package.json"
	if e.storage.contentByPath == nil {
		e.storage.contentByPath = map[string][]byte{}
	}
	e.storage.contentByPath[path] = []byte(`{"document_id":"` + document.ID + `","sender_public_key_pem":"sender-public-key"}`)
	return encryption.EncryptedPackage{DocumentID: document.ID}, path, nil
}
