package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// Qdrant — open-source vector database. Upsert, query, delete vectors;
// manage collections.
func Qdrant() schema.NodeDefinition {
	return schema.NodeDefinition{
		Type:        "ai.qdrant",
		Label:       "Qdrant",
		Group:       "ai",
		Icon:        "Database",
		Description: "Upsert, query, and delete vectors in Qdrant; manage collections.",
		Inputs:      []schema.Port{{ID: "main"}},
		Outputs:     []schema.Port{{ID: "main"}},
		Credentials: []string{"qdrantApi"},
		Params: []schema.ParamSchema{
			{Name: "credential", Label: "Credential", Type: "credential", CredentialType: "qdrantApi"},
			{Name: "operation", Label: "Operation", Type: "select", Required: true, Options: []schema.ParamOption{
				{Label: "Upsert Vectors", Value: "vector:upsert"},
				{Label: "Query Vectors", Value: "vector:query"},
				{Label: "Delete Vectors", Value: "vector:delete"},
				{Label: "List Collections", Value: "collection:list"},
				{Label: "Create Collection", Value: "collection:create"},
				{Label: "Delete Collection", Value: "collection:delete"},
			}},
			{Name: "host", Label: "Qdrant Host URL", Type: "string", Required: true,
				Placeholder: "https://my-cluster.qdrant.io:6333"},
			{Name: "collection", Label: "Collection Name", Type: "string", Required: true},
			{Name: "vector", Label: "Vector Data (JSON)", Type: "json",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"vector:upsert", "vector:query"}},
				Description: "For upsert: [{id, vector, payload?}]; for query: {vector:[...], limit:10}."},
			{Name: "ids", Label: "Point IDs (JSON array)", Type: "json",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"vector:delete"}},
				Description: "Array of point IDs to delete."},
			{Name: "dimension", Label: "Vector Size", Type: "number", Default: 1536,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"collection:create"}}},
			{Name: "distance", Label: "Distance Metric", Type: "select", Default: "Cosine", Options: []schema.ParamOption{
				{Label: "Cosine", Value: "Cosine"}, {Label: "Euclid", Value: "Euclid"}, {Label: "Dot", Value: "Dot"},
			}, ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"collection:create"}}},
		},
		Execute: func(ctx *schema.ExecContext) (schema.NodeResult, error) {
			op := str(ctx.Params, "operation", "vector:query")
			apiKey := getCredStr(ctx, "credential", "apiKey", "")
			host := str(ctx.Params, "host", "")
			collection := str(ctx.Params, "collection", "")

			if host == "" || collection == "" {
				return schema.NodeResult{}, fmt.Errorf("qdrant: host and collection are required")
			}

			base := host + "/collections/" + collection

			switch op {
			case "vector:upsert":
				body := rawParam(ctx.Params, "vector")
				result, err := qdrantCall(apiKey, http.MethodPut, base+"/points?wait=true", body)
				if err != nil {
					return schema.NodeResult{}, err
				}
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {item(result)}}}, nil

			case "vector:query":
				body := rawParam(ctx.Params, "vector")
				result, err := qdrantCall(apiKey, http.MethodPost, base+"/points/search", body)
				if err != nil {
					return schema.NodeResult{}, err
				}
				items := toItems(result, "result")
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": items}}, nil

			case "vector:delete":
				ids := rawParam(ctx.Params, "ids")
				body := map[string]any{"points": ids}
				result, err := qdrantCall(apiKey, http.MethodPost, base+"/points/delete?wait=true", body)
				if err != nil {
					return schema.NodeResult{}, err
				}
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {item(result)}}}, nil

			case "collection:list":
				result, err := qdrantCall(apiKey, http.MethodGet, host+"/collections", nil)
				if err != nil {
					return schema.NodeResult{}, err
				}
				items := toItems(result, "result.collections")
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": items}}, nil

			case "collection:create":
				dim := num(ctx.Params, "dimension", 1536)
				dist := str(ctx.Params, "distance", "Cosine")
				body := map[string]any{
					"vectors": map[string]any{
						"size":     dim,
						"distance": dist,
					},
				}
				result, err := qdrantCall(apiKey, http.MethodPut, host+"/collections/"+collection+"?wait=true", body)
				if err != nil {
					return schema.NodeResult{}, err
				}
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {item(result)}}}, nil

			case "collection:delete":
				result, err := qdrantCall(apiKey, http.MethodDelete, base+"?wait=true", nil)
				if err != nil {
					return schema.NodeResult{}, err
				}
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {item(result)}}}, nil

			default:
				return schema.NodeResult{}, fmt.Errorf("qdrant: unknown operation %q", op)
			}
		},
	}
}

func qdrantCall(apiKey, method, url string, body any) (map[string]any, error) {
	var bodyBytes []byte
	if body != nil {
		bodyBytes, _ = json.Marshal(body)
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("qdrant: %w", err)
	}
	req.Header.Set("api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := aiClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("qdrant: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("qdrant: %d — %s", resp.StatusCode, trunc(string(raw), 400))
	}
	var result map[string]any
	if json.Unmarshal(raw, &result) != nil {
		return map[string]any{"raw": string(raw)}, nil
	}
	return result, nil
}
