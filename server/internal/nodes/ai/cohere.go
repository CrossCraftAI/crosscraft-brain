package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// Cohere — Cohere AI language platform: generate, embed, classify, summarize, chat.
func Cohere() schema.NodeDefinition {
	return schema.NodeDefinition{
		Type:        "ai.cohere",
		Label:       "Cohere",
		Group:       "ai",
		Icon:        "MessageSquare",
		Description: "Generate, embed, classify, summarise, and chat with Cohere models.",
		Inputs:      []schema.Port{{ID: "main"}},
		Outputs:     []schema.Port{{ID: "main"}},
		Credentials: []string{"cohereApi"},
		Params: []schema.ParamSchema{
			{Name: "credential", Label: "Credential", Type: "credential", Required: true, CredentialType: "cohereApi"},
			{Name: "operation", Label: "Operation", Type: "select", Required: true, Options: []schema.ParamOption{
				{Label: "Generate Text", Value: "generate"},
				{Label: "Create Embeddings", Value: "embed"},
				{Label: "Classify Text", Value: "classify"},
				{Label: "Summarise Text", Value: "summarize"},
				{Label: "Chat Completion", Value: "chat"},
			}},
			{Name: "model", Label: "Model", Type: "string", Default: "command-r-plus",
				Placeholder: "command-r-plus / embed-english-v3.0"},
			{Name: "prompt", Label: "Prompt / Text", Type: "expression", Required: true,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"generate", "classify", "summarize", "chat", "embed"}}},
			{Name: "maxTokens", Label: "Max Tokens", Type: "number", Default: 500,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"generate", "chat"}}},
			{Name: "temperature", Label: "Temperature", Type: "number", Default: 0.7,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"generate", "chat"}}},
		},
		Execute: func(ctx *schema.ExecContext) (schema.NodeResult, error) {
			op := str(ctx.Params, "operation", "generate")
			apiKey := getCredStr(ctx, "credential", "apiKey", "")
			model := str(ctx.Params, "model", "command-r-plus")
			prompt := str(ctx.Params, "prompt", "")

			if prompt == "" {
				return schema.NodeResult{}, fmt.Errorf("cohere: prompt is required")
			}

			var endpoint string
			var body map[string]any

			switch op {
			case "generate":
				endpoint = "https://api.cohere.ai/v1/generate"
				body = map[string]any{
					"model":        model,
					"prompt":       prompt,
					"max_tokens":   num(ctx.Params, "maxTokens", 500),
					"temperature":  num(ctx.Params, "temperature", 0.7),
				}
			case "embed":
				endpoint = "https://api.cohere.ai/v1/embed"
				texts := []string{prompt}
				if arr, ok := ctx.Params["prompt"].([]any); ok {
					texts = make([]string, len(arr))
					for i, e := range arr {
						texts[i] = fmt.Sprint(e)
					}
				}
				body = map[string]any{"model": model, "texts": texts,
					"input_type": "search_document"}
			case "classify":
				endpoint = "https://api.cohere.ai/v1/classify"
				body = map[string]any{"model": model, "inputs": []string{prompt}}
			case "summarize":
				endpoint = "https://api.cohere.ai/v1/summarize"
				body = map[string]any{"model": model, "text": prompt}
			case "chat":
				endpoint = "https://api.cohere.ai/v1/chat"
				maxT := num(ctx.Params, "maxTokens", 500)
				temp := num(ctx.Params, "temperature", 0.7)
				body = map[string]any{
					"model":        model,
					"message":      prompt,
					"max_tokens":   maxT,
					"temperature":  temp,
				}
			default:
				return schema.NodeResult{}, fmt.Errorf("cohere: unknown operation %q", op)
			}

			result, err := cohereCall(apiKey, endpoint, body)
			if err != nil {
				return schema.NodeResult{}, err
			}
			return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {item(result)}}}, nil
		},
	}
}

func cohereCall(apiKey, url string, body map[string]any) (map[string]any, error) {
	bodyBytes, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("cohere: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := aiClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cohere: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("cohere: %d — %s", resp.StatusCode, trunc(string(raw), 400))
	}
	var result map[string]any
	if json.Unmarshal(raw, &result) != nil {
		return map[string]any{"raw": string(raw)}, nil
	}
	return result, nil
}
