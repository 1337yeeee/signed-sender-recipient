package mailer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"electronic-digital-signature/internal/domain/model"
)

type MailpitSource struct {
	baseURL string
	client  *http.Client
}

func NewMailpitSource(baseURL string) *MailpitSource {
	return &MailpitSource{
		baseURL: strings.TrimRight(baseURL, "/"),
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (s *MailpitSource) ListMessages(ctx context.Context, query model.MailQuery) ([]model.MailMessage, error) {
	endpoint, err := url.Parse(s.baseURL + "/api/v1/messages")
	if err != nil {
		return nil, err
	}

	values := endpoint.Query()
	values.Set("start", "0")
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	values.Set("limit", strconv.Itoa(limit))
	endpoint.RawQuery = values.Encode()

	var payload struct {
		Messages []mailpitMessageSummary `json:"messages"`
	}
	if err := s.getJSON(ctx, endpoint.String(), &payload); err != nil {
		return nil, err
	}
	if len(payload.Messages) == 0 {
		var uppercase struct {
			Messages []mailpitMessageSummary `json:"Messages"`
		}
		if err := s.getJSON(ctx, endpoint.String(), &uppercase); err != nil {
			return nil, err
		}
		payload.Messages = uppercase.Messages
	}

	messages := make([]model.MailMessage, 0, len(payload.Messages))
	for _, item := range payload.Messages {
		messages = append(messages, item.toModel())
	}

	return messages, nil
}

func (s *MailpitSource) GetMessage(ctx context.Context, messageID string) (*model.MailMessage, error) {
	var payload mailpitMessageDetails
	if err := s.getJSON(ctx, s.baseURL+"/api/v1/message/"+url.PathEscape(messageID), &payload); err != nil {
		return nil, err
	}

	message := payload.toModel()
	return &message, nil
}

func (s *MailpitSource) DownloadAttachment(ctx context.Context, messageID, attachmentID string) ([]byte, error) {
	endpoint := s.baseURL + path.Join("/api/v1/message", url.PathEscape(messageID), "part", url.PathEscape(attachmentID))
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	response, err := s.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request mailpit attachment: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
		return nil, fmt.Errorf("mailpit attachment request failed: %s", strings.TrimSpace(string(body)))
	}

	content, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read mailpit attachment response: %w", err)
	}

	return content, nil
}

func (s *MailpitSource) getJSON(ctx context.Context, endpoint string, out any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}

	response, err := s.client.Do(request)
	if err != nil {
		return fmt.Errorf("request mailpit api: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
		return fmt.Errorf("mailpit api request failed: %s", strings.TrimSpace(string(body)))
	}

	if err := json.NewDecoder(response.Body).Decode(out); err != nil {
		return fmt.Errorf("decode mailpit api response: %w", err)
	}

	return nil
}

type mailpitMessageSummary struct {
	ID      string              `json:"ID"`
	Subject string              `json:"Subject"`
	From    mailpitAddress      `json:"From"`
	To      []mailpitAddress    `json:"To"`
	Created mailpitFlexibleTime `json:"Created"`
}

func (m mailpitMessageSummary) toModel() model.MailMessage {
	return model.MailMessage{
		ID:        m.ID,
		Subject:   m.Subject,
		From:      m.From.toModel(),
		To:        toAddresses(m.To),
		CreatedAt: m.Created.Time,
	}
}

type mailpitMessageDetails struct {
	ID          string              `json:"ID"`
	Subject     string              `json:"Subject"`
	From        mailpitAddress      `json:"From"`
	To          []mailpitAddress    `json:"To"`
	Created     mailpitFlexibleTime `json:"Created"`
	Attachments []mailpitAttachment `json:"Attachments"`
}

func (m mailpitMessageDetails) toModel() model.MailMessage {
	attachments := make([]model.MailAttachment, 0, len(m.Attachments))
	for _, attachment := range m.Attachments {
		attachments = append(attachments, attachment.toModel())
	}

	return model.MailMessage{
		ID:          m.ID,
		Subject:     m.Subject,
		From:        m.From.toModel(),
		To:          toAddresses(m.To),
		CreatedAt:   m.Created.Time,
		Attachments: attachments,
	}
}

type mailpitAddress struct {
	Name    string `json:"Name"`
	Address string `json:"Address"`
	Email   string `json:"Email"`
}

func (a mailpitAddress) toModel() model.MailAddress {
	email := strings.TrimSpace(a.Address)
	if email == "" {
		email = strings.TrimSpace(a.Email)
	}

	return model.MailAddress{
		Name:  strings.TrimSpace(a.Name),
		Email: email,
	}
}

type mailpitAttachment struct {
	PartID      string `json:"PartID"`
	FileName    string `json:"FileName"`
	ContentType string `json:"ContentType"`
	Size        int64  `json:"Size"`
}

func (a mailpitAttachment) toModel() model.MailAttachment {
	return model.MailAttachment{
		ID:          a.PartID,
		FileName:    a.FileName,
		ContentType: a.ContentType,
		Size:        a.Size,
	}
}

type mailpitFlexibleTime struct {
	time.Time
}

func (t *mailpitFlexibleTime) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		t.Time = time.Time{}
		return nil
	}

	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, raw); err == nil {
			t.Time = parsed
			return nil
		}
	}

	return fmt.Errorf("unsupported mailpit time format: %s", raw)
}

func toAddresses(addresses []mailpitAddress) []model.MailAddress {
	result := make([]model.MailAddress, 0, len(addresses))
	for _, address := range addresses {
		result = append(result, address.toModel())
	}

	return result
}
