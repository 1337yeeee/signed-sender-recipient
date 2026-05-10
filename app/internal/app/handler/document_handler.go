package handler

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"electronic-digital-signature/internal/app/dto"
	"electronic-digital-signature/internal/app/usecase"

	"github.com/gin-gonic/gin"
)

type sendDocumentUseCase interface {
	Execute(ctx context.Context, input usecase.SendDocumentInput) (*usecase.SendDocumentResult, error)
}

type verifyDecryptPackageUseCase interface {
	Execute(ctx context.Context, input usecase.VerifyDecryptPackageInput) (*usecase.VerifyDecryptPackageResult, error)
}

type DocumentHandler struct {
	sendDocumentUseCase         sendDocumentUseCase
	verifyDecryptPackageUseCase verifyDecryptPackageUseCase
}

func NewDocumentHandler(
	sendDocumentUseCase sendDocumentUseCase,
	verifyDecryptPackageUseCase verifyDecryptPackageUseCase,
) *DocumentHandler {
	return &DocumentHandler{
		sendDocumentUseCase:         sendDocumentUseCase,
		verifyDecryptPackageUseCase: verifyDecryptPackageUseCase,
	}
}

func (h *DocumentHandler) SendDocument(ctx *gin.Context) {
	if h.sendDocumentUseCase == nil {
		respondError(ctx, http.StatusInternalServerError, "internal_error", "Document sending is not available right now.")
		return
	}

	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		respondError(ctx, http.StatusBadRequest, "document_file_required", "Document file is required.")
		return
	}
	if fileHeader.Size <= 0 {
		respondError(ctx, http.StatusBadRequest, "document_file_required", "Document file is required.")
		return
	}
	if fileHeader.Size > usecase.MaxUploadDocumentSizeBytes {
		respondError(ctx, http.StatusBadRequest, "document_too_large", "Document file exceeds the maximum allowed size.")
		return
	}
	if !strings.EqualFold(fileExtension(fileHeader.Filename), ".docx") {
		respondError(ctx, http.StatusBadRequest, "invalid_document_type", "Document file must have .docx extension.")
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		logRequestError(ctx, "send-document-open-file", err)
		respondError(ctx, http.StatusBadRequest, "document_file_unreadable", "Uploaded document file could not be opened.")
		return
	}
	defer file.Close()

	mimeType := fileHeader.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	}
	if !isAllowedUploadDocumentMIMEType(mimeType) {
		respondError(ctx, http.StatusBadRequest, "invalid_document_type", "Document MIME type is not supported.")
		return
	}

	result, err := h.sendDocumentUseCase.Execute(ctx.Request.Context(), usecase.SendDocumentInput{
		RecipientEmail:   ctx.PostForm("recipient_email"),
		OriginalFileName: fileHeader.Filename,
		MimeType:         mimeType,
		Content:          file,
	})
	if err != nil {
		logRequestError(ctx, "send-document", err)
		switch {
		case strings.Contains(err.Error(), "recipient email is required"):
			respondError(ctx, http.StatusBadRequest, "recipient_email_required", "Recipient email is required.")
		case strings.Contains(err.Error(), ".docx extension"):
			respondError(ctx, http.StatusBadRequest, "invalid_document_type", "Document file must have .docx extension.")
		case strings.Contains(err.Error(), "document file is required"):
			respondError(ctx, http.StatusBadRequest, "document_file_required", "Document file is required.")
		default:
			respondError(ctx, http.StatusBadGateway, "document_send_failed", "Document package could not be sent.")
		}
		return
	}

	respondSuccess(ctx, http.StatusCreated, dto.SendDocumentResponse{
		DocumentID:       result.DocumentID,
		SenderEmail:      result.SenderEmail,
		RecipientEmail:   result.RecipientEmail,
		OriginalFileName: result.OriginalFileName,
		StoredPath:       result.StoredPath,
		PackagePath:      result.PackagePath,
		MimeType:         result.MimeType,
		CreatedAt:        result.CreatedAt.Format(timeRFC3339Nano),
		SignedAt:         result.SignedAt.Format(timeRFC3339Nano),
		SentAt:           result.SentAt.Format(timeRFC3339Nano),
	})
	logRequestInfo(ctx, "send-document", fmt.Sprintf("document_id=%s recipient_email=%s", result.DocumentID, result.RecipientEmail))
}

func (h *DocumentHandler) VerifyDecryptPackage(ctx *gin.Context) {
	if h.verifyDecryptPackageUseCase == nil {
		respondError(ctx, http.StatusInternalServerError, "internal_error", "Package verification is not available right now.")
		return
	}

	packageContent, err := readPackageContent(ctx)
	if err != nil {
		respondError(ctx, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	result, err := h.verifyDecryptPackageUseCase.Execute(ctx.Request.Context(), usecase.VerifyDecryptPackageInput{
		PackageContent: packageContent,
	})
	if err != nil {
		logRequestError(ctx, "verify-decrypt-package", err)
		switch {
		case errors.Is(err, usecase.ErrInvalidSignature):
			respondError(ctx, http.StatusBadRequest, "invalid_signature", "Package signature is invalid.")
		case errors.Is(err, usecase.ErrInvalidEncryptedPackage):
			respondError(ctx, http.StatusBadRequest, "invalid_package", "Encrypted package is invalid.")
		default:
			respondError(ctx, http.StatusBadRequest, "verify_decrypt_failed", "Package could not be verified.")
		}
		return
	}

	respondSuccess(ctx, http.StatusOK, dto.VerifyDecryptPackageResponse{
		Valid: true,
		Metadata: dto.VerifyDecryptPackageMetadata{
			DocumentID:           result.Metadata.DocumentID,
			Version:              result.Metadata.Version,
			SenderEmail:          result.Metadata.SenderEmail,
			SenderKeyFingerprint: result.Metadata.SenderKeyFingerprint,
			SignedAt:             result.Metadata.SignedAt,
			EncryptionAlgorithm:  result.Metadata.EncryptionAlgorithm,
			KeyTransport:         result.Metadata.KeyTransport,
			SignatureAlgorithm:   result.Metadata.SignatureAlgorithm,
			OriginalFileName:     result.Metadata.OriginalFileName,
			MimeType:             result.Metadata.MimeType,
			HashBase64:           result.Metadata.HashBase64,
		},
		DecryptedDocumentBase64: base64.StdEncoding.EncodeToString(result.DecryptedDocument),
	})
}

func readPackageContent(ctx *gin.Context) ([]byte, error) {
	contentType := ctx.GetHeader("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		fileHeader, err := ctx.FormFile("package")
		if err != nil {
			fileHeader, err = ctx.FormFile("file")
			if err != nil {
				return nil, fmt.Errorf("package file is required")
			}
		}

		file, err := fileHeader.Open()
		if err != nil {
			return nil, fmt.Errorf("open package file")
		}
		defer file.Close()

		return io.ReadAll(file)
	}

	content, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		return nil, fmt.Errorf("read package body: %w", err)
	}
	if len(content) == 0 {
		return nil, fmt.Errorf("package json is required")
	}

	return content, nil
}

func isAllowedUploadDocumentMIMEType(mimeType string) bool {
	_, ok := usecase.AllowedUploadDocumentMIMETypes[strings.ToLower(strings.TrimSpace(mimeType))]
	return ok
}

func fileExtension(fileName string) string {
	return strings.ToLower(strings.TrimSpace(filepath.Ext(fileName)))
}

const timeRFC3339Nano = "2006-01-02T15:04:05.999999999Z07:00"
