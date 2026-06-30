package core

import (
	"testing"
)

func TestMarkdownToHTML(t *testing.T) {
	md := "# Hello\n\nThis is **bold** and *italic*.\n\n- Item 1\n- Item 2\n\n[Link](https://example.com)"
	res := mustExec(t, markdownNode, ctxFor(
		items(map[string]any{"markdown": md}),
		map[string]any{"action": "toHtml", "field": "markdown", "destination": "html"}))

	html := res.Outputs["main"][0].JSON["html"].(string)
	if html == "" {
		t.Fatal("expected non-empty HTML output")
	}
	if html[:4] != "<h1>" {
		t.Fatalf("expected h1 heading, got: %s", html[:50])
	}
}

func TestMarkdownToText(t *testing.T) {
	md := "# Title\n\nSome **bold** text and a [link](https://example.com)."
	res := mustExec(t, markdownNode, ctxFor(
		items(map[string]any{"markdown": md}),
		map[string]any{"action": "toText", "field": "markdown", "destination": "text"}))

	text := res.Outputs["main"][0].JSON["text"].(string)
	if text == "" {
		t.Fatal("expected non-empty text output")
	}
	// Should not contain markdown syntax
	if text[:5] != "Title" {
		t.Fatalf("expected 'Title', got: %s", text[:30])
	}
}

func TestMarkdownExtract(t *testing.T) {
	md := "## Section 1\n\n## Section 2\n\n[Google](https://google.com)\n[GitHub](https://github.com)\n\n| A | B |\n|---|---|\n| 1 | 2 |"
	res := mustExec(t, markdownNode, ctxFor(
		items(map[string]any{"markdown": md}),
		map[string]any{"action": "extract", "field": "markdown", "destination": "data"}))

	data := res.Outputs["main"][0].JSON["data"].(map[string]any)
	headings := data["headings"].([]map[string]any)
	if len(headings) != 2 {
		t.Fatalf("expected 2 headings, got %d", len(headings))
	}
	links := data["links"].([]map[string]any)
	if len(links) != 2 {
		t.Fatalf("expected 2 links, got %d", len(links))
	}
	tables := data["tables"].([]map[string]any)
	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}
}

func TestMarkdownCodeBlock(t *testing.T) {
	md := "```go\nfunc main() {}\n```\n\nSome text"
	res := mustExec(t, markdownNode, ctxFor(
		items(map[string]any{"markdown": md}),
		map[string]any{"action": "toHtml", "field": "markdown", "destination": "html"}))

	html := res.Outputs["main"][0].JSON["html"].(string)
	if html == "" {
		t.Fatal("expected non-empty HTML")
	}
	// Should contain <pre><code> block
	if !contains(html, "<pre><code>") {
		t.Fatalf("expected code block in HTML: %s", html[:100])
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
