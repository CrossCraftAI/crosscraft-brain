package aws

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// SESNode provides Amazon SES email sending operations.
func SESNode() schema.NodeDefinition {
	return schema.NodeDefinition{
		Type: "aws.ses", Label: "AWS SES", Group: "integration", Icon: "Mail",
		Description: "Send emails via Amazon Simple Email Service.",
		Inputs:  []schema.Port{{ID: "main"}},
		Outputs: []schema.Port{{ID: "main"}},
		Credentials: []string{"awsIam"},
		Params: []schema.ParamSchema{
			{Name: "credential", Label: "Credential", Type: "credential", Required: true, CredentialType: "awsIam"},
			{Name: "operation", Label: "Operation", Type: "select", Required: true, Options: []schema.ParamOption{
				{Label: "Send Email", Value: "send"},
				{Label: "Send Template Email", Value: "sendTemplate"},
				{Label: "List Identities", Value: "listIdentities"},
				{Label: "Get Send Statistics", Value: "getSendStatistics"},
			}},
			{Name: "from", Label: "From email", Type: "string",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"send", "sendTemplate"}}},
			{Name: "to", Label: "To (comma-separated)", Type: "string",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"send", "sendTemplate"}}},
			{Name: "subject", Label: "Subject", Type: "string",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"send", "sendTemplate"}}},
			{Name: "bodyHtml", Label: "HTML body", Type: "code",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"send", "sendTemplate"}}},
			{Name: "bodyText", Label: "Text body", Type: "code",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"send", "sendTemplate"}}},
			{Name: "cc", Label: "CC (comma-separated)", Type: "string",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"send", "sendTemplate"}}},
			{Name: "bcc", Label: "BCC (comma-separated)", Type: "string",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"send", "sendTemplate"}}},
			{Name: "templateName", Label: "Template name", Type: "string",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"sendTemplate"}}},
			{Name: "templateData", Label: "Template data (JSON)", Type: "json",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"sendTemplate"}}},
		},
		Execute: func(ctx *schema.ExecContext) (schema.NodeResult, error) {
			signer, err := makeSigner(ctx, "ses")
			if err != nil {
				return schema.NodeResult{}, err
			}
			op := asString(ctx.Params["operation"], "")
			region := signer.Region
			host := "email." + region + ".amazonaws.com"
			baseURL := "https://" + host

			switch op {

			case "send":
				return sesSend(ctx, signer, baseURL, false)

			case "sendTemplate":
				return sesSend(ctx, signer, baseURL, true)

			case "listIdentities":
				params := url.Values{"Action": {"ListIdentities"}, "Version": {"2010-12-01"}}
				items, err := sesQuery(signer, baseURL, params)
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": items}}, err

			case "getSendStatistics":
				params := url.Values{"Action": {"GetSendStatistics"}, "Version": {"2010-12-01"}}
				items, err := sesQuery(signer, baseURL, params)
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": items}}, err

			default:
				return schema.NodeResult{}, fmt.Errorf("ses: unknown operation %q", op)
			}
		},
	}
}

func sesSend(ctx *schema.ExecContext, signer *Signer, baseURL string, isTemplate bool) (schema.NodeResult, error) {
	from := asString(ctx.Params["from"], "")
	to := asString(ctx.Params["to"], "")
	subject := asString(ctx.Params["subject"], "")
	htmlBody := asString(ctx.Params["bodyHtml"], "")
	textBody := asString(ctx.Params["bodyText"], "")
	cc := asString(ctx.Params["cc"], "")
	bcc := asString(ctx.Params["bcc"], "")
	if from == "" || to == "" {
		return schema.NodeResult{}, fmt.Errorf("ses send: from and to are required")
	}

	params := url.Values{}
	params.Set("Action", "SendEmail")
	params.Set("Version", "2010-12-01")
	params.Set("Source", from)

	// Destination
	toAddrs := splitCSV(to)
	for i, addr := range toAddrs {
		params.Set(fmt.Sprintf("Destination.ToAddresses.member.%d", i+1), addr)
	}
	if cc != "" {
		for i, addr := range splitCSV(cc) {
			params.Set(fmt.Sprintf("Destination.CcAddresses.member.%d", i+1), addr)
		}
	}
	if bcc != "" {
		for i, addr := range splitCSV(bcc) {
			params.Set(fmt.Sprintf("Destination.BccAddresses.member.%d", i+1), addr)
		}
	}

	// Message
	params.Set("Message.Subject.Data", subject)
	params.Set("Message.Subject.Charset", "UTF-8")
	if htmlBody != "" {
		params.Set("Message.Body.Html.Data", htmlBody)
		params.Set("Message.Body.Html.Charset", "UTF-8")
	}
	if textBody != "" {
		params.Set("Message.Body.Text.Data", textBody)
		params.Set("Message.Body.Text.Charset", "UTF-8")
	}
	if htmlBody == "" && textBody == "" {
		params.Set("Message.Body.Text.Data", subject)
		params.Set("Message.Body.Text.Charset", "UTF-8")
	}

	bodyStr := params.Encode()
	req, err := http.NewRequest("POST", baseURL, strings.NewReader(bodyStr))
	if err != nil {
		return schema.NodeResult{}, err
	}
	req.Host = req.URL.Host
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.ContentLength = int64(len(bodyStr))

	if err := signer.Sign(req, []byte(bodyStr)); err != nil {
		return schema.NodeResult{}, fmt.Errorf("ses: sign: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return schema.NodeResult{}, fmt.Errorf("ses: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return schema.NodeResult{}, fmt.Errorf("ses: %d %s", resp.StatusCode, truncate(string(raw), 500))
	}

	// Parse XML response
	msgID := extractXMLElement(raw, "MessageId")
	reqID := extractXMLElement(raw, "RequestId")

	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{JSON: map[string]any{
		"sent":      true,
		"messageId": msgID,
		"requestId": reqID,
		"from":      from,
		"to":        to,
	}}}}}, nil
}

func sesQuery(signer *Signer, baseURL string, params url.Values) ([]schema.Item, error) {
	bodyStr := params.Encode()
	req, err := http.NewRequest("POST", baseURL, strings.NewReader(bodyStr))
	if err != nil {
		return nil, err
	}
	req.Host = req.URL.Host
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.ContentLength = int64(len(bodyStr))

	if err := signer.Sign(req, []byte(bodyStr)); err != nil {
		return nil, fmt.Errorf("ses: sign: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ses: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("ses: %d %s", resp.StatusCode, truncate(string(raw), 500))
	}

	// Convert XML to JSON-ish items
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		return []schema.Item{{JSON: map[string]any{"raw": string(raw)}}}, nil
	}
	return anyToItems(result), nil
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func extractXMLElement(data []byte, tag string) string {
	open := "<" + tag + ">"
	closeTag := "</" + tag + ">"
	start := bytes.Index(data, []byte(open))
	if start < 0 {
		return ""
	}
	start += len(open)
	end := bytes.Index(data[start:], []byte(closeTag))
	if end < 0 {
		return ""
	}
	return string(data[start : start+end])
}
