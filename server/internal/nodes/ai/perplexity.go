package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// Perplexity — AI-powered search and chat with citations.
func Perplexity() schema.NodeDefinition {
	return schema.NodeDefinition{
		Type:        "ai.perplexity",
		Label:       "Perplexity",
		Group:       "ai",
		Icon:        "Search",
		Description: "Chat completion and web search with Perplexity AI, including citations.",
		Inputs:      []schema.Port{{ID: "main"}},
		Outputs:     []schema.Port{{ID: "main"}},
		Credentials: []string{"perplexityApi"},
		Params: []schema.ParamSchema{
			{Name: "credential", Label: "Credential", Type: "credential", Required: true, CredentialType: "perplexityApi"},
			{Name: "operation", Label: "Operation", Type: "select", Required: true, Options: []schema.ParamOption{
				{Label: "Chat Completion", Value: "chat:completion"},
				{Label: "Search Query", Value: "search:run"},
				{Label: "List Models", Value: "model:list"},
			}},
			{Name: "model", Label: "Model", Type: "string", Default: "sonar-pro",
				Placeholder: "sonar-pro / sonar / sonar-reasoning"},
			{Name: "prompt", Label: "Prompt / Query", Type: "expression", Required: true,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"chat:completion", "search:run"}}},
			{Name: "maxTokens", Label: "Max Tokens", Type: "number", Default: 1000,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"chat:completion", "search:run"}}},
			{Name: "temperature", Label: "Temperature", Type: "number", Default: 0.2,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"chat:completion"}}},
		},
		Execute: func(ctx *schema.ExecContext) (schema.NodeResult, error) {
			op := str(ctx.Params, "operation", "chat:completion")
			apiKey := getCredStr(ctx, "credential", "apiKey", "")
			model := str(ctx.Params, "model", "sonar-pro")

			switch op {
			case "chat:completion":
				prompt := str(ctx.Params, "prompt", "")
				if prompt == "" {
					return schema.NodeResult{}, fmt.Errorf("perplexity: prompt is required")
				}
				body := map[string]any{
					"model": model,
					"messages": []map[string]any{
						{"role": "user", "content": prompt},
					},
					"max_tokens":  num(ctx.Params, "maxTokens", 1000),
					"temperature": num(ctx.Params, "temperature", 0.2),
				}
				result, err := perplexityCall(apiKey, "https://api.perplexity.ai/chat/completions", body)
				if err != nil {
					return schema.NodeResult{}, err
				}
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {item(result)}}}, nil

			case "search:run":
				prompt := str(ctx.Params, "prompt", "")
				if prompt == "" {
					return schema.NodeResult{}, fmt.Errorf("perplexity: prompt is required")
				}
				body := map[string]any{
					"model": model,
					"messages": []map[string]any{
						{"role": "user", "content": prompt},
					},
					"max_tokens": num(ctx.Params, "maxTokens", 1000),
				}
				result, err := perplexityCall(apiKey, "https://api.perplexity.ai/chat/completions", body)
				if err != nil {
					return schema.NodeResult{}, err
				}
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {item(result)}}}, nil

			case "model:list":
				result, err := perplexityCall(apiKey, "https://api.perplexity.ai/models", nil, http.MethodGet)
				if err != nil {
					return schema.NodeResult{}, err
				}
				items := toItems(result, "data")
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": items}}, nil

			default:
				return schema.NodeResult{}, fmt.Errorf("perplexity: unknown operation %q", op)
			}
		},
	}
}

func perplexityCall(apiKey, url string, body map[string]any, method ...string) (map[string]any, error) {
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
		return nil, fmt.Errorf("perplexity: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := aiClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("perplexity: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("perplexity: %d — %s", resp.StatusCode, trunc(string(raw), 400))
	}
	var result map[string]any
	if json.Unmarshal(raw, &result) != nil {
		return map[string]any{"raw": string(raw)}, nil
	}
	return result, nil
}
