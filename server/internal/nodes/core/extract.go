package core

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

func init() { Nodes = append(Nodes, extractNode) }

// extractNode extends the existing extractFromFile node with XML, PDF metadata,
// and ODS (OpenDocument Spreadsheet) support.
var extractNode = schema.NodeDefinition{
	Type: "core.extractFile", Label: "Extract From File (extended)", Group: "transform", Icon: "FileSearch",
	Description: "Extract structured data from XML, PDF (basic), or ODS spreadsheet files.",
	Inputs:  []schema.Port{{ID: "main"}},
	Outputs: []schema.Port{{ID: "main"}},
	Params: []schema.ParamSchema{
		{Name: "format", Label: "Format", Type: "select", Default: "xml", Options: []schema.ParamOption{
			{Label: "XML", Value: "xml"},
			{Label: "PDF (text)", Value: "pdf"},
			{Label: "ODS Spreadsheet", Value: "ods"},
		}},
		{Name: "binaryProperty", Label: "Binary property", Type: "string", Default: "data"},
		{Name: "xmlRootPath", Label: "XML root element path (dot-separated)", Type: "string",
			ShowWhen: &schema.ShowWhen{Param: "format", Equals: []any{"xml"}}},
	},
	Execute: func(ctx *schema.ExecContext) (schema.NodeResult, error) {
		format := asString(ctx.Params["format"], "xml")
		prop := asString(ctx.Params["binaryProperty"], "data")
		xmlPath := asString(ctx.Params["xmlRootPath"], "")

		out := []schema.Item{}
		for _, item := range itemsOrEmpty(ctx.Input) {
			if item.Binary == nil {
				continue
			}
			ref, ok := item.Binary[prop]
			if !ok {
				continue
			}
			data, err := base64.StdEncoding.DecodeString(ref.Data)
			if err != nil {
				return schema.NodeResult{}, fmt.Errorf("extract %s: decode: %w", format, err)
			}

			switch format {
			case "ods":
				parsed, odsErr := extractODS(data)
				if odsErr != nil {
					return schema.NodeResult{}, odsErr
				}
				out = append(out, parsed...)
			case "pdf":
				// Basic PDF text extraction: looks for text between BT/ET markers.
				item := extractPDFText(data)
				out = append(out, item)
			default: // xml
				item, xmlErr := extractXML(data, xmlPath)
				if xmlErr != nil {
					return schema.NodeResult{}, xmlErr
				}
				out = append(out, item)
			}
		}
		return schema.NodeResult{Outputs: map[string][]schema.Item{"main": out}}, nil
	},
}

// ---------------------------------------------------------------------------
// XML extraction
// ---------------------------------------------------------------------------

func extractXML(data []byte, rootPath string) (schema.Item, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))

	var root map[string]any
	var stack []map[string]any

	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return schema.Item{}, fmt.Errorf("xml parse: %w", err)
		}

		switch t := tok.(type) {
		case xml.StartElement:
			el := map[string]any{
				"_name": t.Name.Local,
			}
			// Collect attributes
			for _, attr := range t.Attr {
				el["@"+attr.Name.Local] = attr.Value
			}
			if len(stack) == 0 {
				root = el
			} else {
				parent := stack[len(stack)-1]
				key := t.Name.Local
				if existing, ok := parent[key]; ok {
					// Convert to array if multiple elements with same name
					if arr, isArr := existing.([]any); isArr {
						parent[key] = append(arr, el)
					} else {
						parent[key] = []any{existing, el}
					}
				} else {
					parent[key] = el
				}
			}
			stack = append(stack, el)
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case xml.CharData:
			text := strings.TrimSpace(string(t))
			if text != "" && len(stack) > 0 {
				el := stack[len(stack)-1]
				if existing, ok := el["_text"]; ok {
					el["_text"] = fmt.Sprintf("%v%s", existing, text)
				} else {
					el["_text"] = text
				}
			}
		}
	}
	if root == nil {
		return schema.Item{}, fmt.Errorf("xml: empty document")
	}
	// Navigate to rootPath if specified
	if rootPath != "" {
		parts := strings.Split(rootPath, ".")
		cur := root
		for _, part := range parts {
			if v, ok := cur[part]; ok {
				if m, isMap := v.(map[string]any); isMap {
					cur = m
				} else {
					return schema.Item{JSON: map[string]any{"_path": rootPath, "_value": v}}, nil
				}
			} else {
				return schema.Item{}, fmt.Errorf("xml: path %q not found", rootPath)
			}
		}
		root = cur
	}
	return schema.Item{JSON: root}, nil
}

// ---------------------------------------------------------------------------
// Basic PDF text extraction
// ---------------------------------------------------------------------------

func extractPDFText(data []byte) schema.Item {
	content := string(data)
	// Minimal PDF text extraction: find text between BT (begin text) and ET (end text).
	// Full extraction requires a proper PDF library like github.com/ledongthuc/pdf.
	var texts []string
	for {
		btIdx := strings.Index(content, "BT")
		if btIdx < 0 {
			break
		}
		etIdx := strings.Index(content[btIdx:], "ET")
		if etIdx < 0 {
			break
		}
		block := content[btIdx : btIdx+etIdx+2]
		// Extract text between parentheses inside Tj/TJ operators
		for _, line := range strings.Split(block, "\n") {
			// Look for (...) patterns (PDF text strings)
			for i := 0; i < len(line); i++ {
				if line[i] == '(' {
					j := strings.IndexByte(line[i+1:], ')')
					if j >= 0 {
						texts = append(texts, line[i+1:i+1+j])
						i += j + 1
					}
				}
			}
		}
		content = content[btIdx+2:]
	}
	return schema.Item{JSON: map[string]any{
		"text":     strings.Join(texts, "\n"),
		"textLines": texts,
		"size":     len(data),
	}}
}

// ---------------------------------------------------------------------------
// ODS extraction
// ---------------------------------------------------------------------------

func extractODS(data []byte) ([]schema.Item, error) {
	// ODS files are ZIP archives containing content.xml
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("ods: not a valid ODS/ZIP file: %w", err)
	}

	// Look for content.xml inside the zip
	for _, f := range r.File {
		if f.Name == "content.xml" {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("ods: open content.xml: %w", err)
			}
			defer rc.Close()
			xmlData, err := io.ReadAll(rc)
			if err != nil {
				return nil, fmt.Errorf("ods: read content.xml: %w", err)
			}
			return parseODSContent(xmlData)
		}
	}
	return nil, fmt.Errorf("ods: content.xml not found in archive")
}

func parseODSContent(data []byte) ([]schema.Item, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var items []schema.Item
	var currentRow map[string]any
	var cellIndex int

	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("ods xml: %w", err)
		}

		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "table-row":
				currentRow = map[string]any{}
				cellIndex = 0
			case "table-cell":
				cellIndex++
				// Get cell value from child <text:p> elements
				var texts []string
				deep := decoder
				for {
					inner, err := deep.Token()
					if err != nil {
						break
					}
					if se, ok := inner.(xml.StartElement); ok && se.Name.Local == "p" {
						if cd, err := deep.Token(); err == nil {
							if ch, ok := cd.(xml.CharData); ok {
								texts = append(texts, string(ch))
							}
						}
					}
					if en, ok := inner.(xml.EndElement); ok && en.Name.Local == "table-cell" {
						break
					}
				}
				if currentRow != nil {
					currentRow[fmt.Sprintf("col%d", cellIndex)] = strings.Join(texts, " ")
				}
			}
		case xml.EndElement:
			if t.Name.Local == "table-row" && currentRow != nil {
				items = append(items, schema.Item{JSON: currentRow})
				currentRow = nil
			}
		}
	}
	return items, nil
}
