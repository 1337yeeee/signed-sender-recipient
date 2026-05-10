package docx

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"electronic-digital-signature/internal/domain/model"
)

type Processor struct{}

func NewProcessor() *Processor {
	return &Processor{}
}

func (p *Processor) AddMetadata(content []byte, metadata model.VisualMetadata) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return nil, fmt.Errorf("open docx archive: %w", err)
	}

	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	documentXMLFound := false

	for _, file := range reader.File {
		if err := copyDocxFile(writer, file, metadata, &documentXMLFound); err != nil {
			writer.Close()
			return nil, err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close updated docx archive: %w", err)
	}
	if !documentXMLFound {
		return nil, fmt.Errorf("docx document.xml not found")
	}

	return output.Bytes(), nil
}

func copyDocxFile(writer *zip.Writer, file *zip.File, metadata model.VisualMetadata, documentXMLFound *bool) error {
	header := file.FileHeader
	header.Name = file.Name
	header.Method = zip.Deflate

	target, err := writer.CreateHeader(&header)
	if err != nil {
		return fmt.Errorf("create docx archive entry %q: %w", file.Name, err)
	}

	source, err := file.Open()
	if err != nil {
		return fmt.Errorf("open docx archive entry %q: %w", file.Name, err)
	}
	defer source.Close()

	if file.Name != "word/document.xml" {
		if _, err := io.Copy(target, source); err != nil {
			return fmt.Errorf("copy docx archive entry %q: %w", file.Name, err)
		}
		return nil
	}

	*documentXMLFound = true
	documentXML, err := io.ReadAll(source)
	if err != nil {
		return fmt.Errorf("read word/document.xml: %w", err)
	}

	updatedXML, err := addMetadataParagraph(string(documentXML), metadata)
	if err != nil {
		return err
	}

	if _, err := target.Write([]byte(updatedXML)); err != nil {
		return fmt.Errorf("write updated word/document.xml: %w", err)
	}

	return nil
}

func addMetadataParagraph(documentXML string, metadata model.VisualMetadata) (string, error) {
	bodyEnd := strings.LastIndex(documentXML, "</w:body>")
	if bodyEnd == -1 {
		return "", fmt.Errorf("docx body end tag not found")
	}

	metadataText := fmt.Sprintf(
		"Document ID: %s; Signed at: %s; Sender: %s; Fingerprint: %s; Version: %s; Signed electronically",
		metadata.DocumentID,
		metadata.SignedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		metadata.SenderEmail,
		metadata.SenderFingerprint,
		firstNonEmpty(metadata.Version, "1"),
	)

	return documentXML[:bodyEnd] + paragraph(metadataText) + documentXML[bodyEnd:], nil
}

func paragraph(text string) string {
	var escaped bytes.Buffer
	_ = xml.EscapeText(&escaped, []byte(text))

	return "<w:p><w:r><w:t>" + escaped.String() + "</w:t></w:r></w:p>"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}

	return ""
}
