package container

import (
	"context"
	"electronic-digital-signature/internal/app/config"
	"electronic-digital-signature/internal/app/handler"
	"electronic-digital-signature/internal/app/realtime"
	"electronic-digital-signature/internal/app/usecase"
	"electronic-digital-signature/internal/infra/crypto"
	"electronic-digital-signature/internal/infra/docx"
	"electronic-digital-signature/internal/infra/encryption"
	"electronic-digital-signature/internal/infra/id"
	"electronic-digital-signature/internal/infra/keys"
	"electronic-digital-signature/internal/infra/mailer"
	"electronic-digital-signature/internal/infra/storage"
	"fmt"
)

type BackgroundService interface {
	Run(ctx context.Context)
}

type AppContainer struct {
	CORSAllowedOrigins []string
	IdentityHandler    *handler.IdentityHandler
	DocumentHandler    *handler.DocumentHandler
	InboundHandler     *handler.InboundHandler
	BackgroundServices []BackgroundService
}

func New(cfg config.Config) (*AppContainer, error) {
	appKeys, err := keys.LoadOrCreateKeyPair(
		cfg.App.PrivateKeyPath,
		cfg.App.PublicKeyPath,
		cfg.App.PrivateKeyPEM,
		cfg.App.PublicKeyPEM,
	)
	if err != nil {
		return nil, err
	}
	fingerprint, err := keys.PublicKeyFingerprint(appKeys.PublicKey)
	if err != nil {
		return nil, err
	}
	documentStorage := storage.NewLocalDocumentStorage(cfg.DocumentStorage.Path)
	inboundRepository := storage.NewLocalInboundRepository(cfg.DocumentStorage.Path)
	signatureProvider := crypto.NewECDSASHA256Provider()
	idGenerator := id.NewUUIDGenerator()
	smtpMailer := mailer.NewSMTPMailer(cfg.SMTP)
	eventBroker := realtime.NewBroker()
	sendDocumentUseCase := usecase.NewSendDocumentUseCase(
		documentStorage,
		idGenerator,
		docx.NewProcessor(),
		signatureProvider,
		appKeys.PrivateKey,
		appKeys.PublicKey,
		cfg.App.Email,
		fingerprint,
		encryption.NewDocumentEncryptor(documentStorage),
		smtpMailer,
	)
	verifyDecryptPackageUseCase := usecase.NewVerifyDecryptPackageUseCase(
		encryption.NewAESGCMEncryptor(),
		signatureProvider,
	)

	backgroundServices := make([]BackgroundService, 0, 1)
	if cfg.MailPolling.Enabled {
		var mailSource usecase.MailSource
		switch cfg.MailPolling.Source.Type {
		case "", "mailpit":
			mailSource = mailer.NewMailpitSource(cfg.MailPolling.Source.BaseURL)
		default:
			return nil, fmt.Errorf("unsupported mail source type: %s", cfg.MailPolling.Source.Type)
		}

		messageFilter := usecase.NewCompositeMessageFilter(
			usecase.NewRecipientFilter(cfg.MailPolling.Filter.RecipientEmail),
			usecase.NewSubjectPrefixFilter(cfg.MailPolling.Filter.SubjectPrefix),
			usecase.NewAttachmentSuffixFilter(cfg.MailPolling.Filter.AttachmentSuffix),
		)
		processor := usecase.NewIncomingPackageProcessor(
			mailSource,
			messageFilter,
			inboundRepository,
			documentStorage,
			verifyDecryptPackageUseCase,
			cfg.App.Email,
			cfg.MailPolling.Source.Limit,
			eventBroker,
		)
		backgroundServices = append(backgroundServices, usecase.NewMailPollingWorker(cfg.MailPolling.Interval, processor))
	}

	return &AppContainer{
		CORSAllowedOrigins: cfg.CORS.AllowedOrigins,
		IdentityHandler: handler.NewIdentityHandler(
			cfg.App.Role,
			cfg.App.Email,
			string(appKeys.PublicKey),
			fingerprint,
		),
		DocumentHandler: handler.NewDocumentHandler(
			sendDocumentUseCase,
			verifyDecryptPackageUseCase,
		),
		InboundHandler: handler.NewInboundHandler(
			inboundRepository,
			eventBroker,
			documentStorage,
		),
		BackgroundServices: backgroundServices,
	}, nil
}
