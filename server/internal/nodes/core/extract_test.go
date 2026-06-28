package core

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"testing"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

func TestExtractXML(t *testing.T) {
	xmlData := `<?xml version="1.0"?><root><item name="test">Hello</item><item name="world">World</item></root>`
	b64 := base64.StdEncoding.EncodeToString([]byte(xmlData))

	res := mustExec(t, extractNode, ctxFor([]schema.Item{{
		JSON:   map[string]any{},
		Binary: map[string]schema.BinaryRef{"data": {Data: b64}},
	}}, map[string]any{"format": "xml", "binaryProperty": "data"}))

	out := res.Outputs["main"]
	if len(out) != 1 {
		t.Fatalf("expected 1 item, got %d", len(out))
	}
	root := out[0].JSON
	if root["_name"] != "root" {
		t.Fatalf("expected root element, got %v", root["_name"])
	}
	items, ok := root["item"].([]any)
	if !ok || len(items) != 2 {
		t.Fatalf("expected 2 item elements, got %+v", root["item"])
	}
}

func TestExtractXMLWithPath(t *testing.T) {
	xmlData := `<?xml version="1.0"?><r><a><b>value</b></a></r>`
	b64 := base64.StdEncoding.EncodeToString([]byte(xmlData))

	res := mustExec(t, extractNode, ctxFor([]schema.Item{{
		JSON:   map[string]any{},
		Binary: map[string]schema.BinaryRef{"data": {Data: b64}},
	}}, map[string]any{"format": "xml", "binaryProperty": "data", "xmlRootPath": "a"}))

	out := res.Outputs["main"]
	if out[0].JSON["_name"] != "a" {
		t.Fatalf("expected 'a' element at path, got %v", out[0].JSON["_name"])
	}
}

func TestExtractPDF(t *testing.T) {
	// Minimal PDF content with a text object
	pdfData := `%PDF-1.4
1 0 obj
<< /Type /Catalog /Pages 2 0 R >>
endobj
2 0 obj
<< /Type /Pages /Kids [3 0 R] /Count 1 >>
endobj
3 0 obj
<< /Type /Page /Parent 2 0 R >>
stream
BT
/F1 12 Tf
100 700 Td
(Hello PDF World) Tj
ET
endstream
endobj
xref
0 4
trailer
<< /Size 4 /Root 1 0 R >>
startxref
%%EOF`
	b64 := base64.StdEncoding.EncodeToString([]byte(pdfData))

	res := mustExec(t, extractNode, ctxFor([]schema.Item{{
		JSON:   map[string]any{},
		Binary: map[string]schema.BinaryRef{"data": {Data: b64}},
	}}, map[string]any{"format": "pdf", "binaryProperty": "data"}))

	if res.Outputs["main"][0].JSON["text"] != "Hello PDF World" {
		t.Fatalf("expected extracted text 'Hello PDF World', got %q", res.Outputs["main"][0].JSON["text"])
	}
}

func TestExtractODS(t *testing.T) {
	// Build a minimal ODS (ZIP with content.xml)
	odsData := minimalODS()
	b64 := base64.StdEncoding.EncodeToString(odsData)

	res := mustExec(t, extractNode, ctxFor([]schema.Item{{
		JSON:   map[string]any{},
		Binary: map[string]schema.BinaryRef{"data": {Data: b64}},
	}}, map[string]any{"format": "ods", "binaryProperty": "data"}))

	out := res.Outputs["main"]
	if len(out) == 0 {
		t.Fatal("expected at least 1 row from ODS")
	}
}

func minimalODS() []byte {
	// Build a minimal ODS ZIP archive with content.xml containing table data
	contentXML := `<?xml version="1.0" encoding="UTF-8"?>
<office:document-content xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
  xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0"
  xmlns:table="urn:oasis:names:tc:opendocument:xmlns:table:1.0">
  <office:body>
    <office:spreadsheet>
      <table:table table:name="Sheet1">
        <table:table-row>
          <table:table-cell><text:p>Name</text:p></table:table-cell>
          <table:table-cell><text:p>Age</text:p></table:table-cell>
        </table:table-row>
        <table:table-row>
          <table:table-cell><text:p>Alice</text:p></table:table-cell>
          <table:table-cell><text:p>30</text:p></table:table-cell>
        </table:table-row>
      </table:table>
    </office:spreadsheet>
  </office:body>
</office:document-content>`

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("content.xml")
	w.Write([]byte(contentXML))
	zw.Close()
	return buf.Bytes()
}

var _ = schema.ExecContext{}
