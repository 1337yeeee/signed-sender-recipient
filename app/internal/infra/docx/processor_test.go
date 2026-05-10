package docx

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	"electronic-digital-signature/internal/domain/model"
)

func TestProcessorAddMetadataCreatesHeaderAndReferencesIt(t *testing.T) {
	input := buildTestDocx(t, map[string]string{
		contentTypesPath:  `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="xml" ContentType="application/xml"/></Types>`,
		documentRelsPath:  `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="styles" Target="styles.xml"/></Relationships>`,
		documentXMLPath:   `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:document xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t>Hello</w:t></w:r></w:p><w:sectPr><w:pgSz w:w="11906" w:h="16838"/><w:pgMar w:top="1134" w:right="850" w:bottom="1134" w:left="1701" w:header="708" w:footer="708" w:gutter="0"/></w:sectPr></w:body></w:document>`,
		"word/styles.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><styles xmlns="http://schemas.openxmlformats.org/wordprocessingml/2006/main"/>`,
	})

	processor := NewProcessor()
	output, err := processor.AddMetadata(input, model.VisualMetadata{
		DocumentID:        "f0977f67-b8b2-4413-ad61-58f15037ce34",
		SenderEmail:       "recipient@example.com",
		SenderFingerprint: "6f9fa4028159fc1bb6655eaaf41100d6c0291396cef93aa7826b5840344365a1",
		SignedAt:          time.Date(2026, 5, 10, 18, 41, 47, 0, time.UTC),
		Version:           "2",
	})
	if err != nil {
		t.Fatalf("add metadata: %v", err)
	}

	files := unzipTestDocx(t, output)

	headerXML := files["word/header1.xml"]
	if !strings.Contains(headerXML, `Times New Roman`) {
		t.Fatalf("expected Times New Roman in header XML, got %s", headerXML)
	}
	if !strings.Contains(headerXML, `w:sz w:val="10"`) {
		t.Fatalf("expected 5pt size in header XML, got %s", headerXML)
	}
	if !strings.Contains(headerXML, `f0977f67-b8b2-4413-ad61-58f15037ce34; Signed: 2026-05-10T18:41:47Z; Sender: recipient@example.com; Fingerprint: 6f9fa4028159fc1bb6655eaaf41100d6c0291396cef93aa7826b5840344365a1; V2; Signed electronically`) {
		t.Fatalf("expected formatted metadata in header XML, got %s", headerXML)
	}

	documentXML := files[documentXMLPath]
	for _, token := range []string{
		`<w:headerReference w:type="default" r:id="rId2"/>`,
		`<w:headerReference w:type="first" r:id="rId2"/>`,
		`<w:headerReference w:type="even" r:id="rId2"/>`,
	} {
		if !strings.Contains(documentXML, token) {
			t.Fatalf("expected document.xml to contain %s, got %s", token, documentXML)
		}
	}
	if strings.Contains(documentXML, "Signed electronically") {
		t.Fatalf("expected metadata to be stored in header, not body: %s", documentXML)
	}

	relsXML := files[documentRelsPath]
	if !strings.Contains(relsXML, `Type="`+headerRelationshipType+`"`) || !strings.Contains(relsXML, `Target="header1.xml"`) {
		t.Fatalf("expected header relationship in rels, got %s", relsXML)
	}

	contentTypesXML := files[contentTypesPath]
	if !strings.Contains(contentTypesXML, `PartName="/word/header1.xml"`) || !strings.Contains(contentTypesXML, headerContentType) {
		t.Fatalf("expected header content type override, got %s", contentTypesXML)
	}
}

func buildTestDocx(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)

	for name, content := range files {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatalf("create zip entry %s: %v", name, err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatalf("write zip entry %s: %v", name, err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}

	return buffer.Bytes()
}

func unzipTestDocx(t *testing.T, content []byte) map[string]string {
	t.Helper()

	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("open zip reader: %v", err)
	}

	files := make(map[string]string, len(reader.File))
	for _, file := range reader.File {
		rc, err := file.Open()
		if err != nil {
			t.Fatalf("open zip file %s: %v", file.Name, err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("read zip file %s: %v", file.Name, err)
		}
		files[file.Name] = string(data)
	}

	return files
}
