package usecase

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"electronic-digital-signature/internal/domain/model"
)

const InboundPackageUpdatedEvent = "inbound_package_updated"

type IncomingPackageProcessor struct {
	source      MailSource
	filter      MessageFilter
	repository  InboundPackageRepository
	storage     inboundPackageStorage
	verifier    inboundPackageVerifier
	recipient   string
	fetchLimit  int
	eventStream InboundEventPublisher
}

type inboundPackageVerifier interface {
	Execute(ctx context.Context, input VerifyDecryptPackageInput) (*VerifyDecryptPackageResult, error)
}

func NewIncomingPackageProcessor(
	source MailSource,
	filter MessageFilter,
	repository InboundPackageRepository,
	storage inboundPackageStorage,
	verifier inboundPackageVerifier,
	recipient string,
	fetchLimit int,
	eventStream InboundEventPublisher,
) *IncomingPackageProcessor {
	if fetchLimit <= 0 {
		fetchLimit = 25
	}

	return &IncomingPackageProcessor{
		source:      source,
		filter:      filter,
		repository:  repository,
		storage:     storage,
		verifier:    verifier,
		recipient:   strings.TrimSpace(recipient),
		fetchLimit:  fetchLimit,
		eventStream: eventStream,
	}
}

func (p *IncomingPackageProcessor) Poll(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if p.source == nil || p.repository == nil || p.storage == nil || p.verifier == nil {
		return fmt.Errorf("incoming package processor is not fully configured")
	}

	messages, err := p.source.ListMessages(ctx, model.MailQuery{Limit: p.fetchLimit})
	if err != nil {
		return err
	}

	for _, summary := range messages {
		if err := ctx.Err(); err != nil {
			return err
		}
		existing, err := p.repository.GetByMailMessageID(ctx, summary.ID)
		if err != nil {
			return err
		}
		if existing != nil {
			continue
		}
		if err := p.processMessage(ctx, summary.ID); err != nil {
			log.Printf("incoming-mail process message %s: %v", summary.ID, err)
		}
	}

	return nil
}

func (p *IncomingPackageProcessor) processMessage(ctx context.Context, messageID string) error {
	message, err := p.source.GetMessage(ctx, messageID)
	if err != nil {
		return fmt.Errorf("load message details: %w", err)
	}
	if message == nil {
		return fmt.Errorf("message details are empty")
	}
	if p.filter != nil && !p.filter.Match(*message) {
		return nil
	}

	attachment, err := selectPackageAttachment(message.Attachments)
	if err != nil {
		return err
	}

	rawPackage, err := p.source.DownloadAttachment(ctx, message.ID, attachment.ID)
	if err != nil {
		return fmt.Errorf("download message attachment: %w", err)
	}

	record := model.InboundPackage{
		MailMessageID:   message.ID,
		RecipientEmail:  p.recipient,
		Subject:         message.Subject,
		MailReceivedAt:  normalizeMailTime(message.CreatedAt),
		ProcessedAt:     time.Now().UTC(),
		Status:          model.PackageStatusProcessing,
		PackageFileName: attachment.FileName,
	}

	packagePath, err := p.storage.SaveInboundPackage(ctx, message.ID, attachment.FileName, rawPackage)
	if err != nil {
		return fmt.Errorf("save inbound package: %w", err)
	}
	record.PackagePath = packagePath

	if err := p.saveAndPublish(ctx, record); err != nil {
		return err
	}

	result, err := p.verifier.Execute(ctx, VerifyDecryptPackageInput{PackageContent: rawPackage})
	if err != nil {
		record.Status = model.PackageStatusFailed
		record.ErrorMessage = err.Error()
		record.ProcessedAt = time.Now().UTC()
		return p.saveAndPublish(ctx, record)
	}

	decryptedPath, err := p.storage.SaveDecryptedDocument(ctx, result.Metadata.DocumentID, result.Metadata.OriginalFileName, result.DecryptedDocument)
	if err != nil {
		record.Status = model.PackageStatusFailed
		record.ErrorMessage = fmt.Sprintf("save decrypted document: %v", err)
		record.ProcessedAt = time.Now().UTC()
		return p.saveAndPublish(ctx, record)
	}

	record.Status = model.PackageStatusProcessed
	record.SignatureValid = true
	record.ProcessedAt = time.Now().UTC()
	record.DecryptedDocumentPath = decryptedPath
	record.ErrorMessage = ""
	record.DocumentID = result.Metadata.DocumentID
	record.SenderEmail = result.Metadata.SenderEmail
	record.SenderKeyFingerprint = result.Metadata.SenderKeyFingerprint
	record.SignedAt = result.Metadata.SignedAt
	record.EncryptionAlgorithm = result.Metadata.EncryptionAlgorithm
	record.KeyTransport = result.Metadata.KeyTransport
	record.SignatureAlgorithm = result.Metadata.SignatureAlgorithm
	record.OriginalFileName = result.Metadata.OriginalFileName
	record.MimeType = result.Metadata.MimeType
	record.HashBase64 = result.Metadata.HashBase64

	return p.saveAndPublish(ctx, record)
}

func (p *IncomingPackageProcessor) saveAndPublish(ctx context.Context, record model.InboundPackage) error {
	if err := p.repository.Save(ctx, record); err != nil {
		return err
	}
	if p.eventStream != nil {
		p.eventStream.Publish(InboundPackageEvent{
			Type:      InboundPackageUpdatedEvent,
			Timestamp: time.Now().UTC(),
			Package:   record,
		})
	}

	return nil
}

type MailPollingWorker struct {
	interval  time.Duration
	processor *IncomingPackageProcessor
}

func NewMailPollingWorker(interval time.Duration, processor *IncomingPackageProcessor) *MailPollingWorker {
	if interval <= 0 {
		interval = 5 * time.Second
	}

	return &MailPollingWorker{
		interval:  interval,
		processor: processor,
	}
}

func (w *MailPollingWorker) Run(ctx context.Context) {
	if w == nil || w.processor == nil {
		return
	}

	runPoll := func() {
		if err := w.processor.Poll(ctx); err != nil && ctx.Err() == nil {
			log.Printf("mail-poller: %v", err)
		}
	}

	runPoll()

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runPoll()
		}
	}
}

func selectPackageAttachment(attachments []model.MailAttachment) (model.MailAttachment, error) {
	for _, attachment := range attachments {
		if strings.EqualFold(filepath.Ext(attachment.FileName), ".json") {
			return attachment, nil
		}
	}

	return model.MailAttachment{}, fmt.Errorf("message does not contain a supported package attachment")
}

func normalizeMailTime(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}

	return value.UTC()
}
