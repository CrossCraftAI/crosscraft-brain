package core

import (
	"encoding/base64"
	"testing"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

func TestSpreadsheetReadCSV(t *testing.T) {
	csv := "name,age\nAlice,30\nBob,25"
	b64 := base64.StdEncoding.EncodeToString([]byte(csv))

	res := mustExec(t, spreadsheetNode, ctxFor([]schema.Item{{
		JSON:   map[string]any{},
		Binary: map[string]schema.BinaryRef{"data": {Data: b64, MimeType: "text/csv"}},
	}}, map[string]any{"action": "read", "format": "csv", "binaryProperty": "data", "hasHeader": true}))

	out := res.Outputs["main"]
	if len(out) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(out))
	}
	if out[0].JSON["name"] != "Alice" || out[0].JSON["age"] != "30" {
		t.Fatalf("unexpected row 0: %+v", out[0].JSON)
	}
}

func TestSpreadsheetReadCSVNoHeader(t *testing.T) {
	csv := "Alice,30\nBob,25"
	b64 := base64.StdEncoding.EncodeToString([]byte(csv))

	res := mustExec(t, spreadsheetNode, ctxFor([]schema.Item{{
		JSON:   map[string]any{},
		Binary: map[string]schema.BinaryRef{"data": {Data: b64, MimeType: "text/csv"}},
	}}, map[string]any{"action": "read", "format": "csv", "binaryProperty": "data", "hasHeader": false}))

	out := res.Outputs["main"]
	if out[0].JSON["col0"] != "Alice" || out[0].JSON["col1"] != "30" {
		t.Fatalf("unexpected: %+v", out[0].JSON)
	}
}

func TestSpreadsheetWriteCSV(t *testing.T) {
	res := mustExec(t, spreadsheetNode, ctxFor(
		items(map[string]any{"name": "Alice", "age": 30}),
		map[string]any{"action": "write", "format": "csv", "outputProperty": "data", "fileName": "test.csv"}))

	out := res.Outputs["main"]
	if out[0].JSON["fileName"] != "test.csv" || out[0].JSON["size"] == nil {
		t.Fatalf("unexpected: %+v", out[0].JSON)
	}
	if out[0].Binary["data"].Data == "" {
		t.Fatal("expected binary output")
	}
}

func TestSpreadsheetWriteXLSX(t *testing.T) {
	res := mustExec(t, spreadsheetNode, ctxFor(
		items(map[string]any{"name": "Alice", "age": 30}),
		map[string]any{"action": "write", "format": "xlsx", "outputProperty": "data", "fileName": "test.xlsx"}))

	out := res.Outputs["main"]
	if out[0].JSON["fileName"] != "test.xlsx" {
		t.Fatalf("expected test.xlsx, got %v", out[0].JSON["fileName"])
	}
	if out[0].Binary["data"].Data == "" {
		t.Fatal("expected binary output for xlsx")
	}
}

func TestSpreadsheetReadXLSX(t *testing.T) {
	// First write an XLSX, then read it back
	writeRes := mustExec(t, spreadsheetNode, ctxFor(
		items(map[string]any{"name": "Alice", "age": 30}),
		map[string]any{"action": "write", "format": "xlsx", "outputProperty": "data", "fileName": "test.xlsx",
			"sheetName": "Data"}))

	xlsxB64 := writeRes.Outputs["main"][0].Binary["data"].Data

	readRes := mustExec(t, spreadsheetNode, ctxFor([]schema.Item{{
		JSON:   map[string]any{},
		Binary: map[string]schema.BinaryRef{"data": {Data: xlsxB64, MimeType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"}},
	}}, map[string]any{"action": "read", "format": "xlsx", "binaryProperty": "data", "hasHeader": true,
		"sheetName": "Data"}))

	out := readRes.Outputs["main"]
	if len(out) != 1 {
		t.Fatalf("expected 1 data row, got %d", len(out))
	}
	if out[0].JSON["name"] != "Alice" || out[0].JSON["age"] != "30" {
		t.Fatalf("unexpected row: %+v", out[0].JSON)
	}
}

var _ = schema.ExecContext{}
