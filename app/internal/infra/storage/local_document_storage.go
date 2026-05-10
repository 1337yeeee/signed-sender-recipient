package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type LocalDocumentStorage struct {
	basePath string
}

func NewLocalDocumentStorage(basePath string) *LocalDocumentStorage {
	return &LocalDocumentStorage{basePath: basePath}
}

func (s *LocalDocumentStorage) Save(ctx context.Context, id, originalFileName string, content io.Reader) (string, error) {
	if s.basePath == "" {
		return "", fmt.Errorf("document storage path is not configured")
	}

	if err := os.MkdirAll(s.basePath, 0o755); err != nil {
		return "", fmt.Errorf("create document storage directory: %w", err)
	}

	fileName := id + "_" + sanitizeFileName(originalFileName)
	path := filepath.Join(s.basePath, fileName)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", fmt.Errorf("create stored document file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, readerWithContext{ctx: ctx, reader: content}); err != nil {
		return "", fmt.Errorf("store document file: %w", err)
	}

	return path, nil
}

func (s *LocalDocumentStorage) SaveEncryptedPackage(ctx context.Context, documentID string, content []byte) (string, error) {
	return s.saveBytes(ctx, s.basePath, documentID+"_encrypted_package.json", content)
}

func (s *LocalDocumentStorage) SaveInboundPackage(ctx context.Context, mailMessageID, originalFileName string, content []byte) (string, error) {
	fileName := mailMessageID + "_" + sanitizeFileName(originalFileName)
	return s.saveBytes(ctx, filepath.Join(s.basePath, "inbound", "packages"), fileName, content)
}

func (s *LocalDocumentStorage) SaveDecryptedDocument(ctx context.Context, documentID, originalFileName string, content []byte) (string, error) {
	fileName := documentID + "_" + sanitizeFileName(originalFileName)
	return s.saveBytes(ctx, filepath.Join(s.basePath, "inbound", "documents"), fileName, content)
}

func (s *LocalDocumentStorage) saveBytes(ctx context.Context, directory, fileName string, content []byte) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if s.basePath == "" {
		return "", fmt.Errorf("document storage path is not configured")
	}

	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", fmt.Errorf("create document storage directory: %w", err)
	}

	path := filepath.Join(directory, fileName)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", fmt.Errorf("create stored file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, readerWithContext{ctx: ctx, reader: bytes.NewReader(content)}); err != nil {
		return "", fmt.Errorf("store document file: %w", err)
	}

	return path, nil
}

func (s *LocalDocumentStorage) Read(ctx context.Context, path string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if path == "" {
		return nil, fmt.Errorf("document path is required")
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read document file: %w", err)
	}

	return content, nil
}

func sanitizeFileName(name string) string {
	name = filepath.Base(name)
	name = strings.ReplaceAll(name, string(filepath.Separator), "_")
	if name == "." || name == "" {
		return "document.docx"
	}

	return name
}

type readerWithContext struct {
	ctx    context.Context
	reader io.Reader
}

func (r readerWithContext) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}

	return r.reader.Read(p)
}
