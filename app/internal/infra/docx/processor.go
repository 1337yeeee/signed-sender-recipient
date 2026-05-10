package docx

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"electronic-digital-signature/internal/domain/model"
)

const (
	documentXMLPath        = "word/document.xml"
	documentRelsPath       = "word/_rels/document.xml.rels"
	contentTypesPath       = "[Content_Types].xml"
	headerRelationshipType = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/header"
	headerContentType      = "application/vnd.openxmlformats-officedocument.wordprocessingml.header+xml"
	defaultHeaderBaseName  = "word/header"
	defaultHeaderExtension = ".xml"
)

var sectPrPattern = regexp.MustCompile(`(?s)<w:sectPr\b[^>]*>.*?</w:sectPr>`)

type Processor struct{}

func NewProcessor() *Processor {
	return &Processor{}
}

func (p *Processor) AddMetadata(content []byte, metadata model.VisualMetadata) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return nil, fmt.Errorf("open docx archive: %w", err)
	}

	files := make(map[string][]byte, len(reader.File))
	order := make([]string, 0, len(reader.File))
	for _, file := range reader.File {
		fileContent, err := readZipFile(file)
		if err != nil {
			return nil, err
		}
		files[file.Name] = fileContent
		order = append(order, file.Name)
	}

	documentXML, ok := files[documentXMLPath]
	if !ok {
		return nil, fmt.Errorf("docx document.xml not found")
	}

	headerPath := nextHeaderPath(order)
	headerRelationshipID := nextRelationshipID(string(files[documentRelsPath]))

	updatedDocumentXML, err := attachHeaderReferences(string(documentXML), headerRelationshipID)
	if err != nil {
		return nil, err
	}

	files[documentXMLPath] = []byte(updatedDocumentXML)
	files[documentRelsPath] = []byte(upsertHeaderRelationship(string(files[documentRelsPath]), headerRelationshipID, strings.TrimPrefix(headerPath, "word/")))
	files[contentTypesPath] = []byte(upsertHeaderContentType(string(files[contentTypesPath]), headerPath))
	files[headerPath] = []byte(buildHeaderXML(metadata))

	if !slices.Contains(order, documentRelsPath) {
		order = append(order, documentRelsPath)
	}
	if !slices.Contains(order, contentTypesPath) {
		order = append(order, contentTypesPath)
	}
	order = appendIfMissing(order, headerPath)

	return writeDocx(order, files)
}

func readZipFile(file *zip.File) ([]byte, error) {
	source, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("open docx archive entry %q: %w", file.Name, err)
	}
	defer source.Close()

	content, err := io.ReadAll(source)
	if err != nil {
		return nil, fmt.Errorf("read docx archive entry %q: %w", file.Name, err)
	}

	return content, nil
}

func writeDocx(order []string, files map[string][]byte) ([]byte, error) {
	var output bytes.Buffer
	writer := zip.NewWriter(&output)

	seen := make(map[string]struct{}, len(order))
	for _, name := range order {
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		content, ok := files[name]
		if !ok {
			continue
		}

		header := zip.FileHeader{
			Name:   name,
			Method: zip.Deflate,
		}
		target, err := writer.CreateHeader(&header)
		if err != nil {
			writer.Close()
			return nil, fmt.Errorf("create docx archive entry %q: %w", name, err)
		}
		if _, err := target.Write(content); err != nil {
			writer.Close()
			return nil, fmt.Errorf("write docx archive entry %q: %w", name, err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close updated docx archive: %w", err)
	}

	return output.Bytes(), nil
}

func attachHeaderReferences(documentXML, relationshipID string) (string, error) {
	if strings.TrimSpace(documentXML) == "" {
		return "", fmt.Errorf("document.xml is empty")
	}

	matches := sectPrPattern.FindAllStringIndex(documentXML, -1)
	if len(matches) == 0 {
		return "", fmt.Errorf("docx sectPr not found")
	}

	var builder strings.Builder
	lastIndex := 0
	for _, match := range matches {
		builder.WriteString(documentXML[lastIndex:match[0]])
		builder.WriteString(rewriteSectPr(documentXML[match[0]:match[1]], relationshipID))
		lastIndex = match[1]
	}
	builder.WriteString(documentXML[lastIndex:])

	return builder.String(), nil
}

func rewriteSectPr(sectionXML, relationshipID string) string {
	referenceBlock := fmt.Sprintf(
		`<w:headerReference w:type="default" r:id="%s"/><w:headerReference w:type="first" r:id="%s"/><w:headerReference w:type="even" r:id="%s"/>`,
		relationshipID,
		relationshipID,
		relationshipID,
	)

	openTagEnd := strings.Index(sectionXML, ">")
	if openTagEnd == -1 {
		return sectionXML
	}

	body := sectionXML[openTagEnd+1:]
	body = removeHeaderReferences(body)
	return sectionXML[:openTagEnd+1] + referenceBlock + body
}

func removeHeaderReferences(body string) string {
	patterns := []string{
		`(?s)<w:headerReference\b[^>]*/>`,
		`(?s)<w:headerReference\b[^>]*>.*?</w:headerReference>`,
	}

	result := body
	for _, pattern := range patterns {
		result = regexp.MustCompile(pattern).ReplaceAllString(result, "")
	}

	return result
}

func upsertHeaderRelationship(relsXML, relationshipID, target string) string {
	if strings.TrimSpace(relsXML) == "" {
		relsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"></Relationships>`
	}

	relationshipTag := fmt.Sprintf(`<Relationship Id="%s" Type="%s" Target="%s"/>`, relationshipID, headerRelationshipType, target)
	if strings.Contains(relsXML, `Id="`+relationshipID+`"`) {
		re := regexp.MustCompile(`<Relationship\b[^>]*Id="` + regexp.QuoteMeta(relationshipID) + `"[^>]*/>`)
		return re.ReplaceAllString(relsXML, relationshipTag)
	}

	insertIndex := strings.LastIndex(relsXML, "</Relationships>")
	if insertIndex == -1 {
		return relsXML + relationshipTag
	}

	return relsXML[:insertIndex] + relationshipTag + relsXML[insertIndex:]
}

func upsertHeaderContentType(contentTypesXML, headerPath string) string {
	if strings.TrimSpace(contentTypesXML) == "" {
		contentTypesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"></Types>`
	}

	overrideTag := fmt.Sprintf(`<Override PartName="/%s" ContentType="%s"/>`, headerPath, headerContentType)
	if strings.Contains(contentTypesXML, `PartName="/`+headerPath+`"`) {
		re := regexp.MustCompile(`<Override\b[^>]*PartName="/` + regexp.QuoteMeta(headerPath) + `"[^>]*/>`)
		return re.ReplaceAllString(contentTypesXML, overrideTag)
	}

	insertIndex := strings.LastIndex(contentTypesXML, "</Types>")
	if insertIndex == -1 {
		return contentTypesXML + overrideTag
	}

	return contentTypesXML[:insertIndex] + overrideTag + contentTypesXML[insertIndex:]
}

func buildHeaderXML(metadata model.VisualMetadata) string {
	var escaped bytes.Buffer
	_ = xml.EscapeText(&escaped, []byte(buildMetadataText(metadata)))

	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<w:hdr xmlns:m="http://schemas.openxmlformats.org/officeDocument/2006/math" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing">` +
		`<w:p>` +
		`<w:pPr>` +
		`<w:spacing w:before="0" w:after="0" w:line="240" w:lineRule="auto"/>` +
		`<w:ind w:left="0" w:right="0" w:firstLine="0" w:hanging="0"/>` +
		`<w:jc w:val="left"/>` +
		`</w:pPr>` +
		`<w:r>` +
		`<w:rPr>` +
		`<w:rFonts w:ascii="Times New Roman" w:hAnsi="Times New Roman" w:cs="Times New Roman"/>` +
		`<w:color w:val="000000"/>` +
		`<w:sz w:val="10"/>` +
		`<w:szCs w:val="10"/>` +
		`</w:rPr>` +
		`<w:t xml:space="preserve">` + escaped.String() + `</w:t>` +
		`</w:r>` +
		`</w:p>` +
		`</w:hdr>`
}

func buildMetadataText(metadata model.VisualMetadata) string {
	return fmt.Sprintf(
		"%s; Signed: %s; Sender: %s; Fingerprint: %s; %s; Signed electronically",
		metadata.DocumentID,
		metadata.SignedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		metadata.SenderEmail,
		metadata.SenderFingerprint,
		normalizeVersion(metadata.Version),
	)
}

func normalizeVersion(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		return "V1"
	}
	if strings.HasPrefix(strings.ToUpper(version), "V") {
		return strings.ToUpper(version)
	}

	return "V" + version
}

func nextHeaderPath(existing []string) string {
	used := make(map[int]struct{}, len(existing))
	for _, name := range existing {
		if !strings.HasPrefix(name, defaultHeaderBaseName) || !strings.HasSuffix(name, defaultHeaderExtension) {
			continue
		}
		middle := strings.TrimSuffix(strings.TrimPrefix(name, defaultHeaderBaseName), defaultHeaderExtension)
		number, err := strconv.Atoi(middle)
		if err == nil && number > 0 {
			used[number] = struct{}{}
		}
	}

	for number := 1; ; number++ {
		if _, ok := used[number]; ok {
			continue
		}
		return defaultHeaderBaseName + strconv.Itoa(number) + defaultHeaderExtension
	}
}

func nextRelationshipID(relsXML string) string {
	maxID := 0
	matches := regexp.MustCompile(`Id="rId(\d+)"`).FindAllStringSubmatch(relsXML, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		number, err := strconv.Atoi(match[1])
		if err == nil && number > maxID {
			maxID = number
		}
	}

	return "rId" + strconv.Itoa(maxID+1)
}

func appendIfMissing(values []string, next string) []string {
	if slices.Contains(values, next) {
		return values
	}

	return append(values, next)
}
