package core

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

func init() { Nodes = append(Nodes, spreadsheetNode) }

// spreadsheetNode reads and writes spreadsheet files. It supports CSV natively
// and XLSX via the ZIP+XML approach (xlsx files are ZIP archives of XML).
var spreadsheetNode = schema.NodeDefinition{
	Type: "core.spreadsheetFile", Label: "Spreadsheet File", Group: "transform", Icon: "Table2",
	Description: "Read XLSX/CSV spreadsheet files into JSON, or write JSON items to CSV/XLSX.",
	Inputs:  []schema.Port{{ID: "main"}},
	Outputs: []schema.Port{{ID: "main"}},
	Params: []schema.ParamSchema{
		{Name: "action", Label: "Action", Type: "select", Default: "read", Options: []schema.ParamOption{
			{Label: "Read to JSON", Value: "read"},
			{Label: "Write from JSON", Value: "write"},
			{Label: "Append Rows", Value: "append"},
		}},
		{Name: "format", Label: "File format", Type: "select", Default: "csv", Options: []schema.ParamOption{
			{Label: "CSV", Value: "csv"}, {Label: "XLSX", Value: "xlsx"},
		}},
		{Name: "binaryProperty", Label: "Binary property", Type: "string", Default: "data"},
		{Name: "outputProperty", Label: "Output binary property", Type: "string", Default: "data"},
		{Name: "sheetName", Label: "Sheet name (XLSX only)", Type: "string", Default: "Sheet1",
			ShowWhen: &schema.ShowWhen{Param: "format", Equals: []any{"xlsx"}}},
		{Name: "hasHeader", Label: "Has header row", Type: "boolean", Default: true,
			ShowWhen: &schema.ShowWhen{Param: "action", Equals: []any{"read"}}},
		{Name: "fileName", Label: "Output file name", Type: "string", Default: "spreadsheet.csv",
			ShowWhen: &schema.ShowWhen{Param: "action", Equals: []any{"write", "append"}}},
	},
	Execute: func(ctx *schema.ExecContext) (schema.NodeResult, error) {
		action := asString(ctx.Params["action"], "read")
		format := asString(ctx.Params["format"], "csv")
		prop := asString(ctx.Params["binaryProperty"], "data")
		outProp := asString(ctx.Params["outputProperty"], "data")
		sheetName := asString(ctx.Params["sheetName"], "Sheet1")
		hasHeader := true
		if v, ok := ctx.Params["hasHeader"]; ok {
			hasHeader = isTruthy(v)
		}
		fileName := asString(ctx.Params["fileName"], "spreadsheet."+format)

		switch action {
		case "read":
			return readSpreadsheet(ctx, format, prop, hasHeader, sheetName)
		case "write":
			return writeSpreadsheet(ctx, format, outProp, fileName, sheetName)
		case "append":
			return appendSpreadsheet(ctx, format, prop, outProp, fileName)
		default:
			return schema.NodeResult{}, fmt.Errorf("spreadsheet: unknown action %q", action)
		}
	},
}

// ---------------------------------------------------------------------------
// Read
// ---------------------------------------------------------------------------

func readSpreadsheet(ctx *schema.ExecContext, format, prop string, hasHeader bool, sheetName string) (schema.NodeResult, error) {
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
			return schema.NodeResult{}, fmt.Errorf("spreadsheet read: decode: %w", err)
		}

		switch format {
		case "xlsx":
			rows, xlsxErr := readXLSX(data, sheetName)
			if xlsxErr != nil {
				return schema.NodeResult{}, xlsxErr
			}
			if len(rows) == 0 {
				continue
			}
			if hasHeader {
				headers := rows[0]
				for _, row := range rows[1:] {
					m := map[string]any{}
					for i, h := range headers {
						if i < len(row) {
							m[h] = row[i]
						}
					}
					out = append(out, schema.Item{JSON: m})
				}
			} else {
				for _, row := range rows {
					m := map[string]any{}
					for i, v := range row {
						m[fmt.Sprintf("col%d", i)] = v
					}
					out = append(out, schema.Item{JSON: m})
				}
			}
		default: // csv
			r := csv.NewReader(bytes.NewReader(data))
			r.FieldsPerRecord = -1
			rows, err := r.ReadAll()
			if err != nil {
				return schema.NodeResult{}, fmt.Errorf("spreadsheet read csv: %w", err)
			}
			if hasHeader && len(rows) > 0 {
				headers := rows[0]
				for _, row := range rows[1:] {
					m := map[string]any{}
					for i, h := range headers {
						if i < len(row) {
							m[h] = row[i]
						}
					}
					out = append(out, schema.Item{JSON: m})
				}
			} else {
				for _, row := range rows {
					m := map[string]any{}
					for i, v := range row {
						m[fmt.Sprintf("col%d", i)] = v
					}
					out = append(out, schema.Item{JSON: m})
				}
			}
		}
	}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": out}}, nil
}

// ---------------------------------------------------------------------------
// Write
// ---------------------------------------------------------------------------

func writeSpreadsheet(ctx *schema.ExecContext, format, outProp, fileName, sheetName string) (schema.NodeResult, error) {
	in := itemsOrEmpty(ctx.Input)
	var data []byte
	var mimeType string

	switch format {
	case "xlsx":
		var b bytes.Buffer
		if err := writeXLSX(&b, in, sheetName); err != nil {
			return schema.NodeResult{}, err
		}
		data = b.Bytes()
		mimeType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
		if !strings.HasSuffix(fileName, ".xlsx") {
			fileName += ".xlsx"
		}
	default: // csv
		var b bytes.Buffer
		w := csv.NewWriter(&b)
		headers := unionKeys(in)
		_ = w.Write(headers)
		for _, item := range in {
			row := make([]string, len(headers))
			for i, h := range headers {
				if v, ok := item.JSON[h]; ok {
					row[i] = fmt.Sprintf("%v", v)
				}
			}
			_ = w.Write(row)
		}
		w.Flush()
		if err := w.Error(); err != nil {
			return schema.NodeResult{}, err
		}
		data = b.Bytes()
		mimeType = "text/csv"
		if !strings.HasSuffix(fileName, ".csv") {
			fileName += ".csv"
		}
	}

	item := schema.Item{
		JSON: map[string]any{"fileName": fileName, "mimeType": mimeType, "size": len(data)},
		Binary: map[string]schema.BinaryRef{
			outProp: {Data: base64.StdEncoding.EncodeToString(data), MimeType: mimeType, FileName: fileName},
		},
	}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {item}}}, nil
}

func appendSpreadsheet(ctx *schema.ExecContext, format, prop, outProp, fileName string) (schema.NodeResult, error) {
	// Read existing file, append rows, write back
	readResult, err := readSpreadsheet(ctx, format, prop, false, "")
	if err != nil {
		return schema.NodeResult{}, err
	}
	existing := itemsOrEmpty(readResult.Outputs["main"])

	// Get existing raw data for append
	var allData []byte
	for _, item := range itemsOrEmpty(ctx.Input) {
		if item.Binary != nil {
			if ref, ok := item.Binary[prop]; ok {
				decoded, decErr := base64.StdEncoding.DecodeString(ref.Data)
				if decErr != nil {
					return schema.NodeResult{}, fmt.Errorf("spreadsheet append: decode: %w", decErr)
				}
				allData = decoded
				break
			}
		}
	}

	switch format {
	case "csv":
		var b bytes.Buffer
		if allData != nil {
			b.Write(allData)
			if !bytes.HasSuffix(bytes.TrimSpace(allData), []byte("\n")) {
				b.WriteString("\n")
			}
		}
		in := itemsOrEmpty(ctx.Input)
		// Get headers from already-read data
		existingHeaders := unionKeys(existing)
		w := csv.NewWriter(&b)
		for _, item := range in {
			row := make([]string, len(existingHeaders))
			for i, h := range existingHeaders {
				if v, ok := item.JSON[h]; ok {
					row[i] = fmt.Sprintf("%v", v)
				}
			}
			_ = w.Write(row)
		}
		w.Flush()
		if err := w.Error(); err != nil {
			return schema.NodeResult{}, err
		}
		allData = b.Bytes()

	case "xlsx":
		// For XLSX, re-read all rows, append new ones, re-write
		allRows := make([]map[string]any, 0, len(existing)+len(ctx.Input))
		for _, e := range existing {
			allRows = append(allRows, e.JSON)
		}
		for _, in := range ctx.Input {
			allRows = append(allRows, in.JSON)
		}
		var b bytes.Buffer
		items := make([]schema.Item, len(allRows))
		for i, r := range allRows {
			items[i] = schema.Item{JSON: r}
		}
		if err := writeXLSX(&b, items, ""); err != nil {
			return schema.NodeResult{}, err
		}
		allData = b.Bytes()
	}

	mimeType := "text/csv"
	if format == "xlsx" {
		mimeType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	}
	if !strings.Contains(fileName, ".") {
		fileName += "." + format
	}

	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{
		JSON: map[string]any{"fileName": fileName, "size": len(allData), "appended": len(ctx.Input)},
		Binary: map[string]schema.BinaryRef{
			outProp: {Data: base64.StdEncoding.EncodeToString(allData), MimeType: mimeType, FileName: fileName},
		},
	}}}}, nil
}

// ---------------------------------------------------------------------------
// XLSX read / write via ZIP + XML
// ---------------------------------------------------------------------------

// Minimal XLSX reader: parses shared strings and sheet data.
func readXLSX(data []byte, sheetName string) ([][]string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("xlsx read: not a valid xlsx: %w", err)
	}

	// Read shared strings
	sharedStrings := []string{}
	for _, f := range zr.File {
		if f.Name == "xl/sharedStrings.xml" {
			ss, err := readSharedStrings(f)
			if err != nil {
				return nil, fmt.Errorf("xlsx: shared strings: %w", err)
			}
			sharedStrings = ss
			break
		}
	}

	// Read sheet data
	sheetFile := "xl/worksheets/sheet1.xml"
	if sheetName != "" && sheetName != "Sheet1" {
		// Find the correct sheet file from workbook.xml
		sheetFile = resolveSheetFile(zr.File, sheetName)
	}
	for _, f := range zr.File {
		if f.Name == sheetFile {
			return readSheetData(f, sharedStrings)
		}
	}
	return nil, fmt.Errorf("xlsx: sheet %q not found", sheetName)
}

func readSharedStrings(f *zip.File) ([]string, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	data, _ := io.ReadAll(rc)
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var ss []string
	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if se, ok := tok.(xml.StartElement); ok && se.Name.Local == "t" {
			if cd, err := decoder.Token(); err == nil {
				if ch, ok := cd.(xml.CharData); ok {
					ss = append(ss, string(ch))
				}
			}
		}
	}
	return ss, nil
}

func resolveSheetFile(files []*zip.File, name string) string {
	for _, f := range files {
		if f.Name == "xl/workbook.xml" {
			rc, _ := f.Open()
			if rc == nil {
				continue
			}
			data, _ := io.ReadAll(rc)
			rc.Close()
			decoder := xml.NewDecoder(bytes.NewReader(data))
			sheetIdx := 1
			for {
				tok, err := decoder.Token()
				if err != nil {
					break
				}
				if se, ok := tok.(xml.StartElement); ok && se.Name.Local == "sheet" {
					for _, attr := range se.Attr {
						if attr.Name.Local == "name" && attr.Value == name {
							return fmt.Sprintf("xl/worksheets/sheet%d.xml", sheetIdx)
						}
					}
					sheetIdx++
				}
			}
			break
		}
	}
	return "xl/worksheets/sheet1.xml"
}

func readSheetData(f *zip.File, sharedStrings []string) ([][]string, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	data, _ := io.ReadAll(rc)
	decoder := xml.NewDecoder(bytes.NewReader(data))

	var rows [][]string
	var currentRow []string
	var currentCell string
	var cellType string // "s" = shared string, "" = inline

	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			if len(currentRow) > 0 {
				rows = append(rows, currentRow)
			}
			break
		}
		if err != nil {
			return nil, err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "row":
				currentRow = []string{}
			case "c":
				// Get cell reference and type
				for _, attr := range t.Attr {
					if attr.Name.Local == "t" {
						cellType = attr.Value
					}
				}
				currentCell = ""
			case "v":
				if cd, err := decoder.Token(); err == nil {
					if ch, ok := cd.(xml.CharData); ok {
						currentCell = string(ch)
					}
				}
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "c":
				val := currentCell
				if cellType == "s" {
					idx := 0
					fmt.Sscanf(val, "%d", &idx)
					if idx >= 0 && idx < len(sharedStrings) {
						val = sharedStrings[idx]
					}
				}
				currentRow = append(currentRow, val)
				cellType = ""
			case "row":
				// Pad row to match column count if needed
				rows = append(rows, currentRow)
				currentRow = nil
			}
		}
	}
	return rows, nil
}

// ---------------------------------------------------------------------------
// XLSX write
// ---------------------------------------------------------------------------

func writeXLSX(w io.Writer, items []schema.Item, sheetName string) error {
	if sheetName == "" {
		sheetName = "Sheet1"
	}
	headers := unionKeys(items)
	if len(headers) == 0 {
		headers = []string{"col0"}
	}

	// Build a minimal XLSX as a ZIP archive
	zw := zip.NewWriter(w)
	defer zw.Close()

	// [Content_Types].xml
	contentTypes := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="xml" ContentType="application/xml"/>
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
  <Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
  <Override PartName="/xl/sharedStrings.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sharedStrings+xml"/>
</Types>`
	writeZipEntry(zw, "[Content_Types].xml", []byte(contentTypes))

	// _rels/.rels
	rels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
</Relationships>`
	writeZipEntry(zw, "_rels/.rels", []byte(rels))

	// xl/workbook.xml
	wb := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <sheets><sheet name="%s" sheetId="1" r:id="rId1"/></sheets>
</workbook>`, xmlEscape(sheetName))
	writeZipEntry(zw, "xl/workbook.xml", []byte(wb))

	// xl/_rels/workbook.xml.rels
	wbRels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/sharedStrings" Target="sharedStrings.xml"/>
</Relationships>`
	writeZipEntry(zw, "xl/_rels/workbook.xml.rels", []byte(wbRels))

	// Build shared strings and sheet data
	ssMap := map[string]int{}
	ssList := []string{}
	getSS := func(v string) int {
		if idx, ok := ssMap[v]; ok {
			return idx
		}
		idx := len(ssList)
		ssMap[v] = idx
		ssList = append(ssList, v)
		return idx
	}

	var sheetXML strings.Builder
	sheetXML.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	sheetXML.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`)

	// Header row
	sheetXML.WriteString(`<row r="1">`)
	for i, h := range headers {
		col := colName(i)
		idx := getSS(h)
		sheetXML.WriteString(fmt.Sprintf(`<c r="%s1" t="s"><v>%d</v></c>`, col, idx))
	}
	sheetXML.WriteString(`</row>`)

	// Data rows
	for r, item := range items {
		rowNum := r + 2
		sheetXML.WriteString(fmt.Sprintf(`<row r="%d">`, rowNum))
		for i, h := range headers {
			val := fmt.Sprintf("%v", item.JSON[h])
			col := colName(i)
			idx := getSS(val)
			sheetXML.WriteString(fmt.Sprintf(`<c r="%s%d" t="s"><v>%d</v></c>`, col, rowNum, idx))
		}
		sheetXML.WriteString(`</row>`)
	}
	sheetXML.WriteString(`</sheetData></worksheet>`)
	writeZipEntry(zw, "xl/worksheets/sheet1.xml", []byte(sheetXML.String()))

	// Shared strings
	var ssXML strings.Builder
	ssXML.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	ssXML.WriteString(fmt.Sprintf(`<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" count="%d" uniqueCount="%d">`, len(ssList), len(ssList)))
	for _, s := range ssList {
		ssXML.WriteString(`<si><t>`)
		ssXML.WriteString(xmlEscape(s))
		ssXML.WriteString(`</t></si>`)
	}
	ssXML.WriteString(`</sst>`)
	writeZipEntry(zw, "xl/sharedStrings.xml", []byte(ssXML.String()))

	return nil
}

func writeZipEntry(zw *zip.Writer, name string, data []byte) {
	w, _ := zw.Create(name)
	w.Write(data)
}

func colName(i int) string {
	name := ""
	for i >= 0 {
		name = string(rune('A'+i%26)) + name
		i = i/26 - 1
	}
	return name
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

var _ = sort.Strings // keep sort import
