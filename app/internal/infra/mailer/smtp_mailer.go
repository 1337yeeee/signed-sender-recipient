package mailer

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/smtp"
	"strings"

	"electronic-digital-signature/internal/app/config"
	"electronic-digital-signature/internal/app/usecase"
)

type SMTPMailer struct {
	host     string
	port     string
	user     string
	password string
	from     string
}

func NewSMTPMailer(cfg config.SMTPConfig) *SMTPMailer {
	return &SMTPMailer{
		host:     cfg.Host,
		port:     cfg.Port,
		user:     cfg.User,
		password: cfg.Password,
		from:     cfg.From,
	}
}

func (m *SMTPMailer) SendEmail(ctx context.Context, to []string, subject, body string, attachments []usecase.EmailAttachment) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if m.host == "" || m.port == "" {
		return fmt.Errorf("smtp host and port are required")
	}
	if m.from == "" {
		return fmt.Errorf("smtp from address is required")
	}
	if len(to) == 0 {
		return fmt.Errorf("email recipients are required")
	}

	message, err := buildMIMEMessage(m.from, to, subject, body, attachments)
	if err != nil {
		return err
	}

	var auth smtp.Auth
	if m.user != "" || m.password != "" {
		auth = smtp.PlainAuth("", m.user, m.password, m.host)
	}

	if err := smtp.SendMail(m.host+":"+m.port, auth, m.from, to, message); err != nil {
		return fmt.Errorf("send email via smtp: %w", err)
	}

	return nil
}

func buildMIMEMessage(from string, to []string, subject, body string, attachments []usecase.EmailAttachment) ([]byte, error) {
	boundary, err := randomBoundary()
	if err != nil {
		return nil, err
	}

	var message bytes.Buffer
	writeHeader(&message, "From", from)
	writeHeader(&message, "To", strings.Join(to, ", "))
	writeHeader(&message, "Subject", subject)
	writeHeader(&message, "MIME-Version", "1.0")
	writeHeader(&message, "Content-Type", `multipart/mixed; boundary="`+boundary+`"`)
	message.WriteString("\r\n")

	message.WriteString("--" + boundary + "\r\n")
	writeHeader(&message, "Content-Type", `text/plain; charset="utf-8"`)
	writeHeader(&message, "Content-Transfer-Encoding", "8bit")
	message.WriteString("\r\n")
	message.WriteString(body + "\r\n")

	for _, attachment := range attachments {
		contentType := attachment.ContentType
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		message.WriteString("--" + boundary + "\r\n")
		writeHeader(&message, "Content-Type", contentType)
		writeHeader(&message, "Content-Disposition", `attachment; filename="`+attachment.FileName+`"`)
		writeHeader(&message, "Content-Transfer-Encoding", "base64")
		message.WriteString("\r\n")
		message.WriteString(wrapBase64(attachment.Content))
		message.WriteString("\r\n")
	}

	message.WriteString("--" + boundary + "--\r\n")
	return message.Bytes(), nil
}

func writeHeader(buffer *bytes.Buffer, key, value string) {
	buffer.WriteString(key + ": " + value + "\r\n")
}

func wrapBase64(content []byte) string {
	encoded := base64.StdEncoding.EncodeToString(content)
	if encoded == "" {
		return ""
	}

	var wrapped strings.Builder
	for len(encoded) > 76 {
		wrapped.WriteString(encoded[:76] + "\r\n")
		encoded = encoded[76:]
	}
	wrapped.WriteString(encoded)

	return wrapped.String()
}

func randomBoundary() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate mime boundary: %w", err)
	}

	return fmt.Sprintf("boundary-%x", bytes), nil
}
