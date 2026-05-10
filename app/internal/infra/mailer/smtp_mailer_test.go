package mailer

import (
	"encoding/base64"
	"strings"
	"testing"

	"electronic-digital-signature/internal/app/usecase"
)

func TestBuildMIMEMessageWithAttachment(t *testing.T) {
	message, err := buildMIMEMessage(
		"server@example.com",
		[]string{"recipient@example.com"},
		"Encrypted document",
		"See attachment.",
		[]usecase.EmailAttachment{
			{
				FileName:    "package.json",
				ContentType: "application/json",
				Content:     []byte(`{"document_id":"document-id"}`),
			},
		},
	)
	if err != nil {
		t.Fatalf("build MIME message: %v", err)
	}

	content := string(message)
	if !strings.Contains(content, "Subject: Encrypted document") {
		t.Fatalf("expected subject header, got %s", content)
	}
	if !strings.Contains(content, `Content-Disposition: attachment; filename="package.json"`) {
		t.Fatalf("expected attachment disposition, got %s", content)
	}
	if !strings.Contains(content, "Content-Type: application/json") {
		t.Fatalf("expected attachment content type, got %s", content)
	}
	if !strings.Contains(content, base64.StdEncoding.EncodeToString([]byte(`{"document_id":"document-id"}`))) {
		t.Fatalf("expected base64 attachment content, got %s", content)
	}
}
