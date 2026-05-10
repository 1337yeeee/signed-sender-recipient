package container

import (
	"electronic-digital-signature/internal/app/config"
	"electronic-digital-signature/internal/app/handler"
	"electronic-digital-signature/internal/app/usecase"
	"electronic-digital-signature/internal/infra/crypto"
	"electronic-digital-signature/internal/infra/docx"
	"electronic-digital-signature/internal/infra/encryption"
	"electronic-digital-signature/internal/infra/id"
	"electronic-digital-signature/internal/infra/keys"
	"electronic-digital-signature/internal/infra/mailer"
	"electronic-digital-signature/internal/infra/storage"
)

type AppContainer struct {
	CORSAllowedOrigins []string
	IdentityHandler    *handler.IdentityHandler
	DocumentHandler    *handler.DocumentHandler
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
	signatureProvider := crypto.NewECDSASHA256Provider()
	idGenerator := id.NewUUIDGenerator()
	smtpMailer := mailer.NewSMTPMailer(cfg.SMTP)
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
	}, nil
}
