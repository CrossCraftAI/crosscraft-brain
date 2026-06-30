package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// Pinecone — vector database for AI embeddings: upsert, query, delete vectors;
// manage indexes.
func Pinecone() schema.NodeDefinition {
	return schema.NodeDefinition{
		Type:        "ai.pinecone",
		Label:       "Pinecone",
		Group:       "ai",
		Icon:        "Cone",
		Description: "Upsert, query, and delete vectors in Pinecone; manage indexes.",
		Inputs:      []schema.Port{{ID: "main"}},
		Outputs:     []schema.Port{{ID: "main"}},
		Credentials: []string{"pineconeApi"},
		Params: []schema.ParamSchema{
			{Name: "credential", Label: "Credential", Type: "credential", Required: true, CredentialType: "pineconeApi"},
			{Name: "operation", Label: "Operation", Type: "select", Required: true, Options: []schema.ParamOption{
				{Label: "Upsert Vectors", Value: "vector:upsert"},
				{Label: "Query Vectors", Value: "vector:query"},
				{Label: "Delete Vectors", Value: "vector:delete"},
				{Label: "List Indexes", Value: "index:list"},
				{Label: "Create Index", Value: "index:create"},
				{Label: "Delete Index", Value: "index:delete"},
			}},
			{Name: "indexHost", Label: "Index Host URL", Type: "string", Required: true,
				Placeholder: "https://myindex-abc123.svc.us-east1-aws.pinecone.io",
				Description: "Full host URL of the Pinecone index from the console."},
			{Name: "vector", Label: "Vector Values (JSON array)", Type: "json",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"vector:upsert", "vector:query"}},
				Description: "For upsert: [{id, values, metadata?}]; for query: {vector:[...], topK:10}."},
			{Name: "ids", Label: "Vector IDs (JSON array)", Type: "json",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"vector:delete"}},
				Description: "Array of vector IDs to delete."},
			{Name: "namespace", Label: "Namespace", Type: "string",
				Placeholder: "my-namespace"},
			{Name: "indexName", Label: "Index Name", Type: "string",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"index:create", "index:delete"}}},
			{Name: "dimension", Label: "Dimension", Type: "number", Default: 1536,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"index:create"}}},
			{Name: "metric", Label: "Metric", Type: "select", Default: "cosine", Options: []schema.ParamOption{
				{Label: "Cosine", Value: "cosine"}, {Label: "Euclidean", Value: "euclidean"}, {Label: "Dot Product", Value: "dotproduct"},
			}, ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"index:create"}}},
		},
		Execute: func(ctx *schema.ExecContext) (schema.NodeResult, error) {
			op := str(ctx.Params, "operation", "vector:query")
			apiKey := getCredStr(ctx, "credential", "apiKey", "")
			indexHost := str(ctx.Params, "indexHost", "")
			ns := str(ctx.Params, "namespace", "")

			switch op {
			case "vector:upsert":
				if indexHost == "" {
					return schema.NodeResult{}, fmt.Errorf("pinecone: indexHost is required")
				}
				body := map[string]any{"vectors": rawParam(ctx.Params, "vector")}
				if ns != "" {
					body["namespace"] = ns
				}
				result, err := pineconeCall(apiKey, indexHost+"/vectors/upsert", body)
				if err != nil {
					return schema.NodeResult{}, err
				}
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {item(result)}}}, nil

			case "vector:query":
				if indexHost == "" {
					return schema.NodeResult{}, fmt.Errorf("pinecone: indexHost is required")
				}
				body := rawParam(ctx.Params, "vector")
				if body == nil {
					body = map[string]any{"topK": 10}
				}
				if ns != "" {
					if m, ok := body.(map[string]any); ok {
						m["namespace"] = ns
					}
				}
				result, err := pineconeCall(apiKey, indexHost+"/query", body)
				if err != nil {
					return schema.NodeResult{}, err
				}
				items := toItems(result, "matches")
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": items}}, nil

			case "vector:delete":
				if indexHost == "" {
					return schema.NodeResult{}, fmt.Errorf("pinecone: indexHost is required")
				}
				ids := rawParam(ctx.Params, "ids")
				body := map[string]any{"ids": ids}
				if ns != "" {
					body["namespace"] = ns
				}
				result, err := pineconeCall(apiKey, indexHost+"/vectors/delete", body)
				if err != nil {
					return schema.NodeResult{}, err
				}
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {item(result)}}}, nil

			case "index:list":
				result, err := pineconeCall(apiKey, "https://api.pinecone.io/indexes", nil)
				if err != nil {
					return schema.NodeResult{}, err
				}
				items := toItems(result, "indexes")
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": items}}, nil

			case "index:create":
				name := str(ctx.Params, "indexName", "")
				dim := num(ctx.Params, "dimension", 1536)
				metric := str(ctx.Params, "metric", "cosine")
				if name == "" {
					return schema.NodeResult{}, fmt.Errorf("pinecone: indexName is required")
				}
				body := map[string]any{
					"name":      name,
					"dimension": dim,
					"metric":    metric,
					"spec":      map[string]any{"serverless": map[string]any{"cloud": "aws", "region": "us-east-1"}},
				}
				result, err := pineconeCall(apiKey, "https://api.pinecone.io/indexes", body)
				if err != nil {
					return schema.NodeResult{}, err
				}
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {item(result)}}}, nil

			case "index:delete":
				name := str(ctx.Params, "indexName", "")
				if name == "" {
					return schema.NodeResult{}, fmt.Errorf("pinecone: indexName is required")
				}
				result, err := pineconeCall(apiKey, "https://api.pinecone.io/indexes/"+name,
					nil, http.MethodDelete)
				if err != nil {
					return schema.NodeResult{}, err
				}
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {item(result)}}}, nil

			default:
				return schema.NodeResult{}, fmt.Errorf("pinecone: unknown operation %q", op)
			}
		},
	}
}

func pineconeCall(apiKey, url string, body any, method ...string) (map[string]any, error) {
	m := http.MethodPost
	if len(method) > 0 {
		m = method[0]
	}
	var bodyBytes []byte
	if body != nil {
		bodyBytes, _ = json.Marshal(body)
	}
	req, err := http.NewRequest(m, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("pinecone: %w", err)
	}
	req.Header.Set("Api-Key", apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := aiClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pinecone: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("pinecone: %d — %s", resp.StatusCode, trunc(string(raw), 400))
	}
	var result map[string]any
	if json.Unmarshal(raw, &result) != nil {
		return map[string]any{"raw": string(raw)}, nil
	}
	return result, nil
}
