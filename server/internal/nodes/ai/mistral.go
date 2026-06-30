package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// Mistral — Mistral AI platform: chat completion, embeddings, and model listing.
func Mistral() schema.NodeDefinition {
	return schema.NodeDefinition{
		Type:        "ai.mistral",
		Label:       "Mistral",
		Group:       "ai",
		Icon:        "Wind",
		Description: "Chat completion and embeddings via Mistral AI models.",
		Inputs:      []schema.Port{{ID: "main"}},
		Outputs:     []schema.Port{{ID: "main"}},
		Credentials: []string{"mistralApi"},
		Params: []schema.ParamSchema{
			{Name: "credential", Label: "Credential", Type: "credential", Required: true, CredentialType: "mistralApi"},
			{Name: "operation", Label: "Operation", Type: "select", Required: true, Options: []schema.ParamOption{
				{Label: "Chat Completion", Value: "chat:completion"},
				{Label: "Create Embeddings", Value: "embed:create"},
				{Label: "List Models", Value: "model:list"},
			}},
			{Name: "model", Label: "Model", Type: "string", Default: "mistral-large-latest",
				Placeholder: "mistral-large-latest / mistral-embed"},
			{Name: "prompt", Label: "Prompt / Input", Type: "expression", Required: true,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"chat:completion", "embed:create"}}},
			{Name: "maxTokens", Label: "Max Tokens", Type: "number", Default: 1000,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"chat:completion"}}},
			{Name: "temperature", Label: "Temperature", Type: "number", Default: 0.7,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"chat:completion"}}},
		},
		Execute: func(ctx *schema.ExecContext) (schema.NodeResult, error) {
			op := str(ctx.Params, "operation", "chat:completion")
			apiKey := getCredStr(ctx, "credential", "apiKey", "")
			model := str(ctx.Params, "model", "mistral-large-latest")

			switch op {
			case "chat:completion":
				prompt := str(ctx.Params, "prompt", "")
				if prompt == "" {
					return schema.NodeResult{}, fmt.Errorf("mistral: prompt is required")
				}
				body := map[string]any{
					"model": model,
					"messages": []map[string]any{
						{"role": "user", "content": prompt},
					},
					"max_tokens":  num(ctx.Params, "maxTokens", 1000),
					"temperature": num(ctx.Params, "temperature", 0.7),
				}
				result, err := mistralCall(apiKey, "https://api.mistral.ai/v1/chat/completions", body)
				if err != nil {
					return schema.NodeResult{}, err
				}
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {item(result)}}}, nil

			case "embed:create":
				prompt := str(ctx.Params, "prompt", "")
				if prompt == "" {
					return schema.NodeResult{}, fmt.Errorf("mistral: prompt is required")
				}
				inputs := []string{prompt}
				if arr, ok := ctx.Params["prompt"].([]any); ok {
					inputs = make([]string, len(arr))
					for i, e := range arr {
						inputs[i] = fmt.Sprint(e)
					}
				}
				body := map[string]any{
					"model": model,
					"input": inputs,
				}
				result, err := mistralCall(apiKey, "https://api.mistral.ai/v1/embeddings", body)
				if err != nil {
					return schema.NodeResult{}, err
				}
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {item(result)}}}, nil

			case "model:list":
				result, err := mistralCall(apiKey, "https://api.mistral.ai/v1/models", nil)
				if err != nil {
					return schema.NodeResult{}, err
				}
				items := toItems(result, "data")
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": items}}, nil

			default:
				return schema.NodeResult{}, fmt.Errorf("mistral: unknown operation %q", op)
			}
		},
	}
}

func mistralCall(apiKey, url string, body map[string]any) (map[string]any, error) {
	var bodyBytes []byte
	if body != nil {
		bodyBytes, _ = json.Marshal(body)
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("mistral: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := aiClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mistral: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("mistral: %d — %s", resp.StatusCode, trunc(string(raw), 400))
	}
	var result map[string]any
	if json.Unmarshal(raw, &result) != nil {
		return map[string]any{"raw": string(raw)}, nil
	}
	return result, nil
}
