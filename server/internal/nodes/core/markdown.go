package core

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

func init() { Nodes = append(Nodes, markdownNode) }

// markdownNode converts Markdown to HTML, extracts plain text, or parses
// structured data (tables, headings) from Markdown content.
//
// NOTE: For a full CommonMark/GFM-compliant renderer, add github.com/yuin/goldmark.
// This implementation provides basic regex-based conversion suitable for
// simple Markdown documents.
var markdownNode = schema.NodeDefinition{
	Type: "core.markdown", Label: "Markdown", Group: "transform", Icon: "FileText",
	Description: "Convert Markdown to HTML, extract plain text, or parse tables and headings.",
	Inputs:  []schema.Port{{ID: "main"}},
	Outputs: []schema.Port{{ID: "main"}},
	Params: []schema.ParamSchema{
		{Name: "action", Label: "Action", Type: "select", Default: "toHtml", Options: []schema.ParamOption{
			{Label: "Convert to HTML", Value: "toHtml"},
			{Label: "Convert to Plain Text", Value: "toText"},
			{Label: "Extract Data", Value: "extract"},
		}},
		{Name: "field", Label: "Source field", Type: "string", Required: true, Default: "markdown"},
		{Name: "destination", Label: "Output field", Type: "string", Default: "html",
			Description: "Field name for the converted output."},
	},
	Execute: func(ctx *schema.ExecContext) (schema.NodeResult, error) {
		action := asString(ctx.Params["action"], "toHtml")
		field := asString(ctx.Params["field"], "markdown")
		dest := asString(ctx.Params["destination"], "html")

		out := make([]schema.Item, 0, len(itemsOrEmpty(ctx.Input)))
		for _, item := range itemsOrEmpty(ctx.Input) {
			md := fmt.Sprintf("%v", item.JSON[field])
			m := copyJSON(item.JSON)

			switch action {
			case "toHtml":
				m[dest] = markdownToHTML(md)
			case "toText":
				m[dest] = markdownToText(md)
			case "extract":
				m[dest] = extractMarkdownData(md)
			}
			out = append(out, schema.Item{JSON: m, Binary: item.Binary})
		}
		return schema.NodeResult{Outputs: map[string][]schema.Item{"main": out}}, nil
	},
}

// ---------------------------------------------------------------------------
// Basic Markdown → HTML
// ---------------------------------------------------------------------------

var (
	reHeading  = regexp.MustCompile(`(?m)^(#{1,6})\s+(.+)$`)
	reBold     = regexp.MustCompile(`\*\*(.+?)\*\*`)
	reItalic   = regexp.MustCompile(`\*(.+?)\*`)
	reCode     = regexp.MustCompile("`([^`\n]+)`")
	reLink     = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	reImage    = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	reListItem = regexp.MustCompile(`(?m)^[\-\*\+]\s+(.+)$`)
	reOrdered  = regexp.MustCompile(`(?m)^\d+\.\s+(.+)$`)
	reBlockquote = regexp.MustCompile(`(?m)^>\s?(.*)$`)
	reHr       = regexp.MustCompile(`(?m)^[\-\*\_]{3,}\s*$`)
	reCodeBlock = regexp.MustCompile("(?s)```[^\n]*\n(.*?)```")
)

func markdownToHTML(md string) string {
	// Code blocks first (before other transformations)
	md = reCodeBlock.ReplaceAllStringFunc(md, func(m string) string {
		inner := reCodeBlock.FindStringSubmatch(m)
		if len(inner) > 1 {
			return "<pre><code>" + escapeHTML(strings.TrimRight(inner[1], "\n")) + "</code></pre>"
		}
		return m
	})

	lines := strings.Split(md, "\n")
	var out strings.Builder
	inList := false
	inBlockquote := false

	for i, line := range lines {
		// Empty line
		if strings.TrimSpace(line) == "" {
			if inList {
				out.WriteString("</ul>\n")
				inList = false
			}
			if inBlockquote {
				out.WriteString("</blockquote>\n")
				inBlockquote = false
			}
			continue
		}

		// Horizontal rule
		if reHr.MatchString(line) {
			out.WriteString("<hr>\n")
			continue
		}

		// Heading
		if m := reHeading.FindStringSubmatch(line); m != nil {
			if inList {
				out.WriteString("</ul>\n")
				inList = false
			}
			level := len(m[1])
			text := inlineMarkdown(m[2])
			out.WriteString(fmt.Sprintf("<h%d>%s</h%d>\n", level, text, level))
			continue
		}

		// Unordered list
		if m := reListItem.FindStringSubmatch(line); m != nil {
			if !inList {
				out.WriteString("<ul>\n")
				inList = true
			}
			out.WriteString("<li>" + inlineMarkdown(m[1]) + "</li>\n")
			continue
		}

		// Ordered list
		if m := reOrdered.FindStringSubmatch(line); m != nil {
			if !inList {
				out.WriteString("<ol>\n")
				inList = true
			}
			out.WriteString("<li>" + inlineMarkdown(m[1]) + "</li>\n")
			continue
		}

		// Blockquote
		if m := reBlockquote.FindStringSubmatch(line); m != nil {
			if !inBlockquote {
				out.WriteString("<blockquote>\n")
				inBlockquote = true
			}
			out.WriteString("<p>" + inlineMarkdown(m[1]) + "</p>\n")
			continue
		}

		// Paragraph
		if inList {
			out.WriteString("</ul>\n")
			inList = false
		}
		if inBlockquote {
			out.WriteString("</blockquote>\n")
			inBlockquote = false
		}

		// Check if next line is also text (join paragraphs)
		if i+1 < len(lines) && strings.TrimSpace(lines[i+1]) != "" &&
			!reHeading.MatchString(lines[i+1]) && !reListItem.MatchString(lines[i+1]) &&
			!reOrdered.MatchString(lines[i+1]) && !reBlockquote.MatchString(lines[i+1]) &&
			!reHr.MatchString(lines[i+1]) {
			out.WriteString("<p>" + inlineMarkdown(line) + " ")
			// Accumulate paragraph lines
			for i+1 < len(lines) && strings.TrimSpace(lines[i+1]) != "" &&
				!reHeading.MatchString(lines[i+1]) && !reListItem.MatchString(lines[i+1]) &&
				!reOrdered.MatchString(lines[i+1]) && !reBlockquote.MatchString(lines[i+1]) &&
				!reHr.MatchString(lines[i+1]) {
				i++
				out.WriteString(inlineMarkdown(lines[i]) + " ")
			}
			out.WriteString("</p>\n")
		} else {
			out.WriteString("<p>" + inlineMarkdown(line) + "</p>\n")
		}
	}

	if inList {
		out.WriteString("</ul>\n")
	}
	if inBlockquote {
		out.WriteString("</blockquote>\n")
	}

	return strings.TrimSpace(out.String())
}

func inlineMarkdown(text string) string {
	text = reImage.ReplaceAllString(text, `<img src="$2" alt="$1">`)
	text = reLink.ReplaceAllString(text, `<a href="$2">$1</a>`)
	text = reBold.ReplaceAllString(text, `<strong>$1</strong>`)
	text = reItalic.ReplaceAllString(text, `<em>$1</em>`)
	text = reCode.ReplaceAllString(text, `<code>$1</code>`)
	return text
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// ---------------------------------------------------------------------------
// Markdown → Plain Text
// ---------------------------------------------------------------------------

var reMDTag = regexp.MustCompile(`[*_~` + "`" + `#>\[\]!|\\]`)

func markdownToText(md string) string {
	// Remove code blocks
	md = reCodeBlock.ReplaceAllString(md, "")
	// Remove inline formatting markers
	text := reMDTag.ReplaceAllString(md, "")
	// Remove link URLs: [text](url) → text
	text = reLink.ReplaceAllString(text, "$1")
	// Remove image syntax
	text = reImage.ReplaceAllString(text, "$1")
	// Remove list markers
	text = reListItem.ReplaceAllString(text, "$1")
	text = reOrdered.ReplaceAllString(text, "$1")
	// Remove heading markers
	text = reHeading.ReplaceAllString(text, "$2")
	// Remove blockquote markers
	text = reBlockquote.ReplaceAllString(text, "$1")
	// Collapse multiple blank lines
	reMultiNewline := regexp.MustCompile(`\n{3,}`)
	text = reMultiNewline.ReplaceAllString(text, "\n\n")
	return strings.TrimSpace(text)
}

// ---------------------------------------------------------------------------
// Extract structured data from Markdown
// ---------------------------------------------------------------------------

var reMDTable = regexp.MustCompile(`(?s)\|(.+)\|\n\|[-\s|:]+\|\n((?:\|.+\|\n?)*)`)

func extractMarkdownData(md string) map[string]any {
	result := map[string]any{}

	// Extract headings
	headings := []map[string]any{}
	for _, m := range reHeading.FindAllStringSubmatch(md, -1) {
		headings = append(headings, map[string]any{
			"level": len(m[1]),
			"text":  strings.TrimSpace(m[2]),
		})
	}
	result["headings"] = headings

	// Extract links
	links := []map[string]any{}
	for _, m := range reLink.FindAllStringSubmatch(md, -1) {
		links = append(links, map[string]any{
			"text": m[1],
			"url":  m[2],
		})
	}
	result["links"] = links

	// Extract tables
	tables := []map[string]any{}
	for _, tableMatch := range reMDTable.FindAllStringSubmatch(md, -1) {
		if len(tableMatch) < 3 {
			continue
		}
		// Parse header
		headerCells := splitTableRow(tableMatch[1])
		var rows []map[string]any
		// Parse data rows
		for _, row := range strings.Split(strings.TrimSpace(tableMatch[2]), "\n") {
			cells := splitTableRow(row)
			if len(cells) == 0 {
				continue
			}
			rowMap := map[string]any{}
			for i, cell := range cells {
				if i < len(headerCells) && headerCells[i] != "" {
					rowMap[strings.TrimSpace(headerCells[i])] = strings.TrimSpace(cell)
				} else {
					rowMap[fmt.Sprintf("col%d", i)] = strings.TrimSpace(cell)
				}
			}
			rows = append(rows, rowMap)
		}
		tables = append(tables, map[string]any{"rows": rows})
	}
	result["tables"] = tables

	// Extract code blocks
	codes := []string{}
	for _, m := range reCodeBlock.FindAllStringSubmatch(md, -1) {
		if len(m) > 1 {
			codes = append(codes, strings.TrimSpace(m[1]))
		}
	}
	result["codeBlocks"] = codes

	return result
}

func splitTableRow(row string) []string {
	row = strings.Trim(row, "|")
	cells := strings.Split(row, "|")
	result := make([]string, len(cells))
	for i, c := range cells {
		result[i] = strings.TrimSpace(c)
	}
	return result
}
