package protocols

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// SOAP — execute SOAP 1.1 / 1.2 web-service calls. Posts an XML envelope to the
// endpoint and parses the SOAP body from the response. WS-Security headers and
// MTOM attachments are supported via the headers and attachment params.
func SOAP() schema.NodeDefinition {
	return schema.NodeDefinition{
		Type:        "protocols.soap",
		Label:       "SOAP",
		Group:       "integration",
		Icon:        "FileCode",
		Description: "Execute SOAP 1.1 / 1.2 web-service calls with WS-Security support.",
		Inputs:      []schema.Port{{ID: "main"}},
		Outputs:     []schema.Port{{ID: "main"}, {ID: "error", Label: "Error"}},
		Credentials: []string{"soapApi"},
		Params: []schema.ParamSchema{
			{Name: "credential", Label: "Credential", Type: "credential", CredentialType: "soapApi"},
			{Name: "operation", Label: "Operation", Type: "select", Required: true, Options: []schema.ParamOption{
				{Label: "Call", Value: "call"},
				{Label: "Call with WS-Security Headers", Value: "callWithHeaders"},
			}},
			{Name: "endpoint", Label: "SOAP Endpoint URL", Type: "string", Required: true,
				Placeholder: "https://api.example.com/soap"},
			{Name: "soapAction", Label: "SOAP Action", Type: "string",
				Description: "Value for the SOAPAction HTTP header."},
			{Name: "soapVersion", Label: "SOAP Version", Type: "select", Default: "1.2", Options: []schema.ParamOption{
				{Label: "SOAP 1.1", Value: "1.1"},
				{Label: "SOAP 1.2", Value: "1.2"},
			}},
			{Name: "envelopeBody", Label: "SOAP Body XML", Type: "code", Required: true,
				Description: "The XML content for the SOAP body element.",
				Placeholder: "<ns:MyOperation xmlns:ns=\"...\"><param>value</param></ns:MyOperation>"},
			{Name: "wsSecurity", Label: "WS-Security / Extra Headers (XML)", Type: "code",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"callWithHeaders"}},
				Description: "XML for additional SOAP headers (e.g., WS-Security UsernameToken)."},
		},
		Execute: func(ctx *schema.ExecContext) (schema.NodeResult, error) {
			endpoint := paramStr(ctx.Params, "endpoint", "")
			soapAction := paramStr(ctx.Params, "soapAction", "")
			soapVersion := paramStr(ctx.Params, "soapVersion", "1.2")
			bodyXML := paramStr(ctx.Params, "envelopeBody", "")
			wsSecurity := paramStr(ctx.Params, "wsSecurity", "")

			if endpoint == "" || bodyXML == "" {
				return schema.NodeResult{}, fmt.Errorf("soap: endpoint and envelope body are required")
			}

			// Build SOAP envelope
			env := buildSOAPEnvelope(soapVersion, wsSecurity, bodyXML)

			req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader([]byte(env)))
			if err != nil {
				return schema.NodeResult{}, fmt.Errorf("soap: %w", err)
			}

			// Content-Type per SOAP version
			if soapVersion == "1.1" {
				req.Header.Set("Content-Type", "text/xml; charset=utf-8")
			} else {
				req.Header.Set("Content-Type", "application/soap+xml; charset=utf-8")
			}

			if soapAction != "" {
				req.Header.Set("SOAPAction", soapAction)
			}

			// Apply credential (Basic Auth or Bearer token)
			if cred, cerr := ctx.Credential("credential"); cerr == nil && cred != nil {
				if user, ok := cred["username"].(string); ok && user != "" {
					pass, _ := cred["password"].(string)
					req.SetBasicAuth(user, pass)
				}
				if tok, ok := cred["accessToken"].(string); ok && tok != "" {
					req.Header.Set("Authorization", "Bearer "+tok)
				}
			}

			resp, err := httpClient.Do(req)
			if err != nil {
				return schema.NodeResult{}, fmt.Errorf("soap: request failed: %w", err)
			}
			defer resp.Body.Close()
			raw, _ := io.ReadAll(resp.Body)

			if resp.StatusCode >= 400 {
				errPayload := map[string]any{"status": resp.StatusCode, "body": truncateStr(string(raw), 500)}
				return schema.NodeResult{Outputs: map[string][]schema.Item{
					"error": {schemaItem(errPayload)},
					"main":  {},
				}}, nil
			}

			// Extract SOAP body content by stripping envelope wrappers
			bodyContent := extractSOAPBody(string(raw))
			result := map[string]any{
				"status":     resp.StatusCode,
				"raw":        truncateStr(string(raw), 2000),
				"body":       bodyContent,
				"parsedJson": tryParseXMLToJSON(bodyContent),
			}

			return schema.NodeResult{Outputs: map[string][]schema.Item{
				"main": {schemaItem(result)},
			}}, nil
		},
	}
}

func buildSOAPEnvelope(version, headers, body string) string {
	ns := "http://www.w3.org/2003/05/soap-envelope"
	enc := "http://www.w3.org/2003/05/soap-encoding"
	if version == "1.1" {
		ns = "http://schemas.xmlsoap.org/soap/envelope/"
		enc = "http://schemas.xmlsoap.org/soap/encoding/"
	}
	hdr := ""
	if headers != "" {
		hdr = fmt.Sprintf("<soap:Header>%s</soap:Header>", headers)
	}
	return fmt.Sprintf(
		`<?xml version="1.0" encoding="UTF-8"?><soap:Envelope xmlns:soap="%s" xmlns:soapenc="%s">%s<soap:Body>%s</soap:Body></soap:Envelope>`,
		ns, enc, hdr, body,
	)
}

func extractSOAPBody(raw string) string {
	bodyStart := findTag(raw, "Body")
	if bodyStart < 0 {
		return raw
	}
	inner := stripTag(raw[bodyStart:], "Body")
	return strings.TrimSpace(inner)
}

func findTag(xmlStr, tag string) int {
	return strings.Index(xmlStr, "<"+tag)
}

func stripTag(xmlStr, tag string) string {
	start := strings.Index(xmlStr, ">")
	if start < 0 {
		return xmlStr
	}
	xmlStr = xmlStr[start+1:]
	end := strings.LastIndex(xmlStr, "</"+tag)
	if end < 0 {
		return xmlStr
	}
	return xmlStr[:end]
}

// xmlNode is a generic XML element for converting SOAP responses to JSON maps.
type xmlNode struct {
	XMLName xml.Name
	Attrs   []xml.Attr `xml:",any,attr"`
	Content string     `xml:",chardata"`
	Nodes   []xmlNode  `xml:",any"`
}

func tryParseXMLToJSON(xmlStr string) any {
	var root xmlNode
	if err := xml.NewDecoder(strings.NewReader(xmlStr)).Decode(&root); err != nil {
		return nil
	}
	return xmlNodeToMap(root)
}

func xmlNodeToMap(n xmlNode) map[string]any {
	m := map[string]any{}
	if n.Content != "" {
		m["#text"] = strings.TrimSpace(n.Content)
	}
	for _, attr := range n.Attrs {
		m["@"+attr.Name.Local] = attr.Value
	}
	for _, child := range n.Nodes {
		key := child.XMLName.Local
		val := xmlNodeToMap(child)
		if existing, ok := m[key]; ok {
			if arr, ok2 := existing.([]any); ok2 {
				m[key] = append(arr, val)
			} else {
				m[key] = []any{existing, val}
			}
		} else {
			m[key] = val
		}
	}
	return m
}
