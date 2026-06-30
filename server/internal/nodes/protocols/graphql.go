package protocols

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

// GraphQL — execute GraphQL queries and mutations against any GraphQL endpoint.
// Supports variables, operation name, and custom headers. Auth: optional Bearer
// token or API key header via graphqlApi credential.
func GraphQL() schema.NodeDefinition {
	return schema.NodeDefinition{
		Type:        "protocols.graphql",
		Label:       "GraphQL",
		Group:       "integration",
		Icon:        "GitGraph",
		Description: "Execute GraphQL queries, mutations, and subscriptions against any GraphQL endpoint.",
		Inputs:      []schema.Port{{ID: "main"}},
		Outputs:     []schema.Port{{ID: "main"}},
		Credentials: []string{"graphqlApi"},
		Params: []schema.ParamSchema{
			{Name: "credential", Label: "Credential", Type: "credential", CredentialType: "graphqlApi"},
			{Name: "operation", Label: "Operation", Type: "select", Required: true, Options: []schema.ParamOption{
				{Label: "Query", Value: "query"},
				{Label: "Mutation", Value: "mutation"},
			}},
			{Name: "endpoint", Label: "GraphQL Endpoint URL", Type: "string", Required: true,
				Placeholder: "https://api.example.com/graphql"},
			{Name: "query", Label: "GraphQL Query", Type: "code", Required: true,
				Placeholder: "query { users { id name } }"},
			{Name: "variables", Label: "Variables (JSON)", Type: "json",
				Description: "Optional variables for the GraphQL operation."},
			{Name: "operationName", Label: "Operation Name", Type: "string",
				Description: "Name of the operation to execute (for multi-operation documents)."},
			{Name: "headers", Label: "Additional Headers (JSON)", Type: "json",
				Description: "Extra HTTP headers as key-value pairs."},
		},
		Execute: func(ctx *schema.ExecContext) (schema.NodeResult, error) {
			endpoint, _ := ctx.Params["endpoint"].(string)
			query, _ := ctx.Params["query"].(string)
			if endpoint == "" || query == "" {
				return schema.NodeResult{}, fmt.Errorf("graphql: endpoint and query are required")
			}

			body := map[string]any{"query": query}
			if v := ctx.Params["variables"]; v != nil {
				body["variables"] = v
			}
			if v := ctx.Params["operationName"]; v != nil && v.(string) != "" {
				body["operationName"] = v
			}

			bodyBytes, _ := json.Marshal(body)
			req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
			if err != nil {
				return schema.NodeResult{}, fmt.Errorf("graphql: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "application/json")

			// Apply credential (optional Bearer token or API key)
			if cred, cerr := ctx.Credential("credential"); cerr == nil && cred != nil {
				if tok, ok := cred["accessToken"].(string); ok && tok != "" {
					req.Header.Set("Authorization", "Bearer "+tok)
				}
				if key, ok := cred["apiKey"].(string); ok && key != "" {
					req.Header.Set("X-API-Key", key)
				}
			}

			// Apply custom headers
			if rawHeaders := ctx.Params["headers"]; rawHeaders != nil {
				var hdrs map[string]string
				switch v := rawHeaders.(type) {
				case map[string]any:
					for k, val := range v {
						if s, ok := val.(string); ok {
							req.Header.Set(k, s)
						}
					}
				case string:
					_ = json.Unmarshal([]byte(v), &hdrs)
					for k, val := range hdrs {
						req.Header.Set(k, val)
					}
				}
			}

			resp, err := httpClient.Do(req)
			if err != nil {
				return schema.NodeResult{}, fmt.Errorf("graphql: request failed: %w", err)
			}
			defer resp.Body.Close()
			raw, _ := io.ReadAll(resp.Body)

			if resp.StatusCode >= 400 {
				return schema.NodeResult{}, fmt.Errorf("graphql: %d — %s", resp.StatusCode, truncateStr(string(raw), 400))
			}

			var result map[string]any
			if err := json.Unmarshal(raw, &result); err != nil {
				return schema.NodeResult{}, fmt.Errorf("graphql: bad response: %w", err)
			}

			// Surface either "data" or the full response.
			if data, ok := result["data"]; ok {
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {schemaItem(data)}}}, nil
			}
			return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {schemaItem(result)}}}, nil
		},
	}
}

func schemaItem(v any) schema.Item {
	if m, ok := v.(map[string]any); ok {
		return schema.Item{JSON: m}
	}
	return schema.Item{JSON: map[string]any{"value": v}}
}

func truncateStr(s string, n int) string {
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}
