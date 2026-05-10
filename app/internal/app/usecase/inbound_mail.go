package usecase

import (
	"context"
	"time"

	"electronic-digital-signature/internal/domain/model"
)

type MailSource interface {
	ListMessages(ctx context.Context, query model.MailQuery) ([]model.MailMessage, error)
	GetMessage(ctx context.Context, messageID string) (*model.MailMessage, error)
	DownloadAttachment(ctx context.Context, messageID, attachmentID string) ([]byte, error)
}

type MessageFilter interface {
	Match(message model.MailMessage) bool
}

type inboundPackageStorage interface {
	SaveInboundPackage(ctx context.Context, mailMessageID, originalFileName string, content []byte) (string, error)
	SaveDecryptedDocument(ctx context.Context, documentID, originalFileName string, content []byte) (string, error)
}

type InboundPackageRepository interface {
	Save(ctx context.Context, pkg model.InboundPackage) error
	GetByMailMessageID(ctx context.Context, mailMessageID string) (*model.InboundPackage, error)
	List(ctx context.Context) ([]model.InboundPackage, error)
}

type InboundEventPublisher interface {
	Publish(event InboundPackageEvent)
}

type InboundPackageEvent struct {
	Type      string               `json:"type"`
	Timestamp time.Time            `json:"timestamp"`
	Package   model.InboundPackage `json:"package"`
}
