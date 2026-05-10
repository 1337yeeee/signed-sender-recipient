package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"electronic-digital-signature/internal/app/dto"
	"electronic-digital-signature/internal/app/usecase"
	"electronic-digital-signature/internal/domain/model"

	"github.com/gin-gonic/gin"
)

type inboundPackageRepository interface {
	GetByMailMessageID(ctx context.Context, mailMessageID string) (*model.InboundPackage, error)
	List(ctx context.Context) ([]model.InboundPackage, error)
}

type inboundEventSubscriber interface {
	Subscribe(buffer int) (<-chan usecase.InboundPackageEvent, func())
}

type inboundFileStorage interface {
	Read(ctx context.Context, path string) ([]byte, error)
}

type InboundHandler struct {
	repository inboundPackageRepository
	events     inboundEventSubscriber
	storage    inboundFileStorage
}

func NewInboundHandler(repository inboundPackageRepository, events inboundEventSubscriber, storage inboundFileStorage) *InboundHandler {
	return &InboundHandler{
		repository: repository,
		events:     events,
		storage:    storage,
	}
}

func (h *InboundHandler) ListPackages(ctx *gin.Context) {
	if h.repository == nil {
		respondSuccess(ctx, http.StatusOK, []dto.InboundPackageResponse{})
		return
	}

	packages, err := h.repository.List(ctx.Request.Context())
	if err != nil {
		logRequestError(ctx, "list-inbound-packages", err)
		respondError(ctx, http.StatusInternalServerError, "internal_error", "Inbound packages could not be loaded.")
		return
	}

	response := make([]dto.InboundPackageResponse, 0, len(packages))
	for _, pkg := range packages {
		response = append(response, inboundPackageDTO(pkg))
	}

	respondSuccess(ctx, http.StatusOK, response)
}

func (h *InboundHandler) StreamEvents(ctx *gin.Context) {
	if h.events == nil {
		respondError(ctx, http.StatusServiceUnavailable, "events_unavailable", "Live events are not available.")
		return
	}

	events, unsubscribe := h.events.Subscribe(16)
	defer unsubscribe()

	writer := ctx.Writer
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("Connection", "keep-alive")
	writer.Header().Set("X-Accel-Buffering", "no")
	writer.WriteHeader(http.StatusOK)

	flusher, ok := writer.(http.Flusher)
	if !ok {
		respondError(ctx, http.StatusInternalServerError, "streaming_unsupported", "Streaming is not supported.")
		return
	}

	writeSSE(writer, "ready", gin.H{
		"connected_at": time.Now().UTC().Format(timeRFC3339Nano),
	})
	flusher.Flush()

	keepAlive := time.NewTicker(20 * time.Second)
	defer keepAlive.Stop()

	for {
		select {
		case <-ctx.Request.Context().Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			payload := dto.InboundPackageEventResponse{
				Type:      event.Type,
				Timestamp: event.Timestamp.UTC().Format(timeRFC3339Nano),
				Package:   inboundPackageDTO(event.Package),
			}
			writeSSE(writer, "inbound-package", payload)
			flusher.Flush()
		case <-keepAlive.C:
			writeSSE(writer, "ping", gin.H{
				"timestamp": time.Now().UTC().Format(timeRFC3339Nano),
			})
			flusher.Flush()
		}
	}
}

func (h *InboundHandler) DownloadFile(ctx *gin.Context) {
	if h.repository == nil || h.storage == nil {
		respondError(ctx, http.StatusServiceUnavailable, "download_unavailable", "Inbound file download is not available.")
		return
	}

	mailMessageID := strings.TrimSpace(ctx.Param("mailMessageID"))
	kind := strings.TrimSpace(strings.ToLower(ctx.Query("kind")))
	if mailMessageID == "" {
		respondError(ctx, http.StatusBadRequest, "mail_message_id_required", "Mail message id is required.")
		return
	}
	if kind != "package" && kind != "document" {
		respondError(ctx, http.StatusBadRequest, "invalid_kind", "Download kind must be package or document.")
		return
	}

	pkg, err := h.repository.GetByMailMessageID(ctx.Request.Context(), mailMessageID)
	if err != nil {
		logRequestError(ctx, "download-inbound-file", err)
		respondError(ctx, http.StatusInternalServerError, "internal_error", "Inbound package could not be loaded.")
		return
	}
	if pkg == nil {
		respondError(ctx, http.StatusNotFound, "package_not_found", "Inbound package was not found.")
		return
	}

	filePath, fileName, contentType, err := resolveInboundDownload(*pkg, kind)
	if err != nil {
		switch {
		case errors.Is(err, errInboundFileMissing):
			respondError(ctx, http.StatusNotFound, "file_not_found", err.Error())
		default:
			respondError(ctx, http.StatusBadRequest, "invalid_download", err.Error())
		}
		return
	}

	content, err := h.storage.Read(ctx.Request.Context(), filePath)
	if err != nil {
		logRequestError(ctx, "download-inbound-file-read", err)
		respondError(ctx, http.StatusNotFound, "file_not_found", "Stored file could not be read.")
		return
	}

	ctx.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fileName))
	ctx.Data(http.StatusOK, contentType, content)
}

func inboundPackageDTO(pkg model.InboundPackage) dto.InboundPackageResponse {
	return dto.InboundPackageResponse{
		MailMessageID:              pkg.MailMessageID,
		RecipientEmail:             pkg.RecipientEmail,
		Subject:                    pkg.Subject,
		MailReceivedAt:             formatTimeOrEmpty(pkg.MailReceivedAt),
		ProcessedAt:                formatTimeOrEmpty(pkg.ProcessedAt),
		Status:                     string(pkg.Status),
		PackageFileName:            pkg.PackageFileName,
		PackagePath:                pkg.PackagePath,
		DecryptedDocumentPath:      pkg.DecryptedDocumentPath,
		SignatureValid:             pkg.SignatureValid,
		ErrorMessage:               pkg.ErrorMessage,
		DocumentID:                 pkg.DocumentID,
		SenderEmail:                pkg.SenderEmail,
		SenderPublicKeyFingerprint: pkg.SenderKeyFingerprint,
		SignedAt:                   pkg.SignedAt,
		EncryptionAlgorithm:        pkg.EncryptionAlgorithm,
		KeyTransport:               pkg.KeyTransport,
		SignatureAlgorithm:         pkg.SignatureAlgorithm,
		OriginalFileName:           pkg.OriginalFileName,
		MimeType:                   pkg.MimeType,
		HashBase64:                 pkg.HashBase64,
	}
}

func formatTimeOrEmpty(value time.Time) string {
	if value.IsZero() {
		return ""
	}

	return value.UTC().Format(timeRFC3339Nano)
}

func writeSSE(writer gin.ResponseWriter, event string, data any) {
	content, err := json.Marshal(data)
	if err != nil {
		return
	}

	_, _ = fmt.Fprintf(writer, "event: %s\n", event)
	_, _ = fmt.Fprintf(writer, "data: %s\n\n", content)
}

var errInboundFileMissing = errors.New("requested inbound file is not available")

func resolveInboundDownload(pkg model.InboundPackage, kind string) (string, string, string, error) {
	switch kind {
	case "package":
		if strings.TrimSpace(pkg.PackagePath) == "" {
			return "", "", "", errInboundFileMissing
		}
		fileName := pkg.PackageFileName
		if fileName == "" {
			fileName = filepath.Base(pkg.PackagePath)
		}
		return pkg.PackagePath, fileName, "application/json; charset=utf-8", nil
	case "document":
		if strings.TrimSpace(pkg.DecryptedDocumentPath) == "" {
			return "", "", "", errInboundFileMissing
		}
		fileName := pkg.OriginalFileName
		if fileName == "" {
			fileName = filepath.Base(pkg.DecryptedDocumentPath)
		}
		contentType := pkg.MimeType
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		return pkg.DecryptedDocumentPath, fileName, contentType, nil
	default:
		return "", "", "", fmt.Errorf("unsupported download kind")
	}
}
