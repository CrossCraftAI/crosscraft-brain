package aws

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

var httpClient = &http.Client{Timeout: 60 * time.Second}

// Nodes returns the full AWS integration node pack.
func Nodes() []schema.NodeDefinition {
	return []schema.NodeDefinition{
		S3Node(),
		SESNode(),
		SQSNode(),
		LambdaNode(),
		DynamoDBNode(),
	}
}

// --- helpers ---------------------------------------------------------------

func asString(v any, def string) string {
	if v == nil {
		return def
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func asInt(v any, def int) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	}
	return def
}

// awsDo signs and executes an HTTP request with SigV4, then parses the response.
func awsDo(signer *Signer, method, url string, body []byte) ([]schema.Item, error) {
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("aws: build request: %w", err)
	}
	req.Host = req.URL.Host
	if body != nil {
		req.ContentLength = int64(len(body))
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	if err := signer.Sign(req, body); err != nil {
		return nil, fmt.Errorf("aws: sign: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("aws: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		var errDoc map[string]any
		if json.Unmarshal(raw, &errDoc) == nil {
			if msg, ok := errDoc["message"].(string); ok {
				return nil, fmt.Errorf("aws: %d %s", resp.StatusCode, msg)
			}
			if msg, ok := errDoc["__type"].(string); ok {
				return nil, fmt.Errorf("aws: %d %s", resp.StatusCode, msg)
			}
		}
		return nil, fmt.Errorf("aws: %d %s", resp.StatusCode, truncate(string(raw), 300))
	}

	if len(bytes.TrimSpace(raw)) == 0 {
		return []schema.Item{{JSON: map[string]any{"success": true}}}, nil
	}

	var root any
	if err := json.Unmarshal(raw, &root); err != nil {
		return []schema.Item{{JSON: map[string]any{"raw": string(raw)}}}, nil
	}
	return anyToItems(root), nil
}

func anyToItems(v any) []schema.Item {
	switch t := v.(type) {
	case []any:
		out := make([]schema.Item, 0, len(t))
		for _, e := range t {
			if m, ok := e.(map[string]any); ok {
				out = append(out, schema.Item{JSON: m})
			}
		}
		if len(out) > 0 {
			return out
		}
		return []schema.Item{{JSON: map[string]any{"list": v}}}
	case map[string]any:
		// Check for nested lists (like S3 ListObjectsResult)
		for _, val := range t {
			if arr, ok := val.([]any); ok {
				out := make([]schema.Item, 0, len(arr))
				for _, e := range arr {
					if m, ok := e.(map[string]any); ok {
						out = append(out, schema.Item{JSON: m})
					}
				}
				if len(out) > 0 {
					return out
				}
			}
		}
		return []schema.Item{{JSON: t}}
	default:
		return []schema.Item{{JSON: map[string]any{"value": v}}}
	}
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

// makeSigner extracts AWS credentials from ExecContext and builds a Signer.
func makeSigner(ctx *schema.ExecContext, service string) (*Signer, error) {
	cred, err := ctx.Credential("credential")
	if err != nil {
		return nil, fmt.Errorf("aws: credential: %w", err)
	}
	accessKey, _ := cred["accessKey"].(string)
	secretKey, _ := cred["secretKey"].(string)
	region, _ := cred["region"].(string)
	if accessKey == "" || secretKey == "" || region == "" {
		return nil, fmt.Errorf("aws: accessKey, secretKey, and region are required in credential")
	}
	return &Signer{AccessKey: accessKey, SecretKey: secretKey, Region: region, Service: service}, nil
}
