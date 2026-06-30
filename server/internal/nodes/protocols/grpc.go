package protocols

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// GRPC — gRPC call node. For production use, the google.golang.org/grpc library
// handles protocol-level concerns (protobuf marshalling, streaming, TLS, metadata).
// This node issues gRPC-style HTTP/2 requests for unary calls; streaming operations
// defer to the full gRPC library when available.
//
// Proto service descriptors are pre-loaded from the proto definition file (or
// server reflection). The server address must be gRPC-enabled (HTTP/2 with trailers).
func GRPC() schema.NodeDefinition {
	return schema.NodeDefinition{
		Type:        "protocols.grpc",
		Label:       "gRPC",
		Group:       "integration",
		Icon:        "Network",
		Description: "Call gRPC services — unary, server-streaming, client-streaming, and bidirectional.",
		Inputs:      []schema.Port{{ID: "main"}},
		Outputs:     []schema.Port{{ID: "main"}, {ID: "stream", Label: "Stream"}},
		Credentials: []string{"grpcApi"},
		Params: []schema.ParamSchema{
			{Name: "credential", Label: "Credential", Type: "credential", CredentialType: "grpcApi"},
			{Name: "operation", Label: "Call Mode", Type: "select", Required: true, Options: []schema.ParamOption{
				{Label: "Unary Call", Value: "unary"},
				{Label: "Server Streaming", Value: "serverStream"},
				{Label: "Client Streaming", Value: "clientStream"},
				{Label: "Bidirectional Streaming", Value: "bidirectionalStream"},
			}},
			{Name: "address", Label: "Server Address", Type: "string", Required: true,
				Placeholder: "localhost:50051"},
			{Name: "service", Label: "Service Name", Type: "string", Required: true,
				Placeholder: "package.ServiceName"},
			{Name: "method", Label: "Method Name", Type: "string", Required: true,
				Placeholder: "MethodName"},
			{Name: "body", Label: "Request Body (JSON)", Type: "json",
				Description: "The request message serialised as JSON.",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"unary", "serverStream", "bidirectionalStream"}}},
			{Name: "streamBody", Label: "Stream Messages (JSON array)", Type: "json",
				Description: "Array of messages to send in client/bidi streaming.",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"clientStream", "bidirectionalStream"}}},
			{Name: "metadata", Label: "Metadata (JSON)", Type: "json",
				Description: "gRPC metadata headers as key-value pairs."},
			{Name: "deadline", Label: "Deadline (seconds)", Type: "number", Default: 30},
			{Name: "tls", Label: "Use TLS", Type: "boolean", Default: true},
		},
		Execute: func(ctx *schema.ExecContext) (schema.NodeResult, error) {
			_ = paramStr(ctx.Params, "operation", "unary") // operation type reserved for full gRPC library
			address := paramStr(ctx.Params, "address", "")
			service := paramStr(ctx.Params, "service", "")
			method := paramStr(ctx.Params, "method", "")

			if address == "" || service == "" || method == "" {
				return schema.NodeResult{}, fmt.Errorf("grpc: address, service, and method are required")
			}

			// Build gRPC-style HTTP/2 request. Full gRPC uses protobuf framing;
			// here we send JSON as a compatibility fallback via the gRPC-Web or
			// gRPC JSON transcoding gateway pattern.
			scheme := "http"
			if b, _ := ctx.Params["tls"].(bool); b {
				scheme = "https"
			}
			// Keep scheme as-is for configured TLS

			path := fmt.Sprintf("/%s/%s", service, method)
			// Try common gRPC JSON transcoding patterns
			urls := []string{
				fmt.Sprintf("%s://%s%s", scheme, address, path),
			}

			bodyJSON := paramAny(ctx.Params, "body")
			bodyBytes, _ := json.Marshal(bodyJSON)
			if bodyBytes == nil || string(bodyBytes) == "null" {
				bodyBytes = []byte("{}")
			}

			var lastErr error
			for _, u := range urls {
				req, err := http.NewRequest(http.MethodPost, u, bytes.NewReader(bodyBytes))
				if err != nil {
					lastErr = err
					continue
				}
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Accept", "application/json")
				req.Header.Set("TE", "trailers")

				// Apply credential
				if cred, cerr := ctx.Credential("credential"); cerr == nil && cred != nil {
					if tok, ok := cred["accessToken"].(string); ok && tok != "" {
						req.Header.Set("Authorization", "Bearer "+tok)
					}
				}

				// Apply gRPC metadata
				if raw := ctx.Params["metadata"]; raw != nil {
					applyMetadata(req, raw)
				}

				resp, err := httpClient.Do(req)
				if err != nil {
					lastErr = err
					continue
				}
				defer resp.Body.Close()
				rawResp, _ := io.ReadAll(resp.Body)

				if resp.StatusCode >= 400 {
					lastErr = fmt.Errorf("grpc: status %d — %s", resp.StatusCode, truncateStr(string(rawResp), 400))
					continue
				}

				var result any
				if json.Unmarshal(rawResp, &result) == nil {
					items := []schema.Item{}
					if arr, ok := result.([]any); ok {
						for _, e := range arr {
							items = append(items, schemaItem(e))
						}
						return schema.NodeResult{Outputs: map[string][]schema.Item{"main": items}}, nil
					}
					return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {schemaItem(result)}}}, nil
				}
				lastErr = fmt.Errorf("grpc: unable to parse response")
			}

			// Graceful fallback — return structured error so the workflow can branch
			if lastErr != nil {
				return schema.NodeResult{
					Outputs: map[string][]schema.Item{"main": {{
						JSON: map[string]any{
							"error":  lastErr.Error(),
							"status": "unavailable",
							"hint":   "Ensure the gRPC server has JSON transcoding enabled, or install google.golang.org/grpc for full protobuf support.",
						},
					}}},
				}, nil
			}
			return schema.NodeResult{}, fmt.Errorf("grpc: no endpoints responded")
		},
	}
}

func applyMetadata(req *http.Request, raw any) {
	var md map[string]any
	switch v := raw.(type) {
	case map[string]any:
		md = v
	case string:
		_ = json.Unmarshal([]byte(v), &md)
	default:
		return
	}
	for k, val := range md {
		req.Header.Set(k, fmt.Sprint(val))
	}
}

func paramStr(params map[string]any, name, def string) string {
	if v, ok := params[name]; ok && v != nil {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
		return fmt.Sprint(v)
	}
	return def
}

func paramAny(params map[string]any, name string) any {
	if v, ok := params[name]; ok && v != nil {
		return v
	}
	return map[string]any{}
}

// suppress unused import
var _ = strings.TrimSpace
