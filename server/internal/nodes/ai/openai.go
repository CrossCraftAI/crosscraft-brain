package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// OpenAI — OpenAI API node: chat, completions, embeddings, DALL·E images, Whisper
// audio transcription/translation, moderation, and model listing.
func OpenAI() schema.NodeDefinition {
	return schema.NodeDefinition{
		Type:        "ai.openai",
		Label:       "OpenAI",
		Group:       "ai",
		Icon:        "Bot",
		Description: "Chat, completions, embeddings, DALL·E images, Whisper audio, moderation — via OpenAI API.",
		Inputs:      []schema.Port{{ID: "main"}},
		Outputs:     []schema.Port{{ID: "main"}, {ID: "images", Label: "Images"}},
		Credentials: []string{"openaiApi"},
		Params: []schema.ParamSchema{
			{Name: "credential", Label: "Credential", Type: "credential", Required: true, CredentialType: "openaiApi"},
			{Name: "operation", Label: "Operation", Type: "select", Required: true, Options: []schema.ParamOption{
				{Label: "Chat Completion", Value: "chat:completion"},
				{Label: "Text Completion", Value: "completion:create"},
				{Label: "Create Embeddings", Value: "embedding:create"},
				{Label: "Generate Image (DALL·E)", Value: "image:generate"},
				{Label: "Transcribe Audio (Whisper)", Value: "audio:transcribe"},
				{Label: "Translate Audio (Whisper)", Value: "audio:translate"},
				{Label: "Moderation Check", Value: "moderation:check"},
				{Label: "List Models", Value: "model:list"},
			}},
			{Name: "model", Label: "Model", Type: "string", Default: "gpt-4o",
				Placeholder: "gpt-4o / gpt-4o-mini / text-embedding-3-small / dall-e-3 / whisper-1"},
			{Name: "prompt", Label: "Prompt / Input", Type: "expression", Required: true,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"chat:completion", "completion:create", "embedding:create", "image:generate", "moderation:check"}}},
			{Name: "systemPrompt", Label: "System Prompt", Type: "expression",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"chat:completion"}},
				Description: "System-level instruction that sets the behaviour of the assistant."},
			{Name: "maxTokens", Label: "Max Tokens", Type: "number", Default: 1000,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"chat:completion", "completion:create"}}},
			{Name: "temperature", Label: "Temperature", Type: "number", Default: 0.7,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"chat:completion", "completion:create"}}},
			{Name: "imageSize", Label: "Image Size", Type: "select", Default: "1024x1024", Options: []schema.ParamOption{
				{Label: "1024×1024", Value: "1024x1024"},
				{Label: "1792×1024", Value: "1792x1024"},
				{Label: "1024×1792", Value: "1024x1792"},
			}, ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"image:generate"}}},
			{Name: "audioFile", Label: "Audio File URL or Base64", Type: "string",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"audio:transcribe", "audio:translate"}},
				Description: "Public URL or base64-encoded audio data."},
			{Name: "language", Label: "Language (ISO 639-1)", Type: "string",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"audio:transcribe"}},
				Description: "Optional language code for transcription."},
		},
		Execute: func(ctx *schema.ExecContext) (schema.NodeResult, error) {
			op := str(ctx.Params, "operation", "chat:completion")
			apiKey := getCredStr(ctx, "credential", "apiKey", "")
			model := str(ctx.Params, "model", "gpt-4o")

			switch op {
			case "chat:completion":
				prompt := str(ctx.Params, "prompt", "")
				if prompt == "" {
					return schema.NodeResult{}, fmt.Errorf("openai: prompt is required")
				}
				messages := []map[string]any{}
				if sys := str(ctx.Params, "systemPrompt", ""); sys != "" {
					messages = append(messages, map[string]any{"role": "system", "content": sys})
				}
				messages = append(messages, map[string]any{"role": "user", "content": prompt})
				body := map[string]any{
					"model":       model,
					"messages":    messages,
					"max_tokens":  num(ctx.Params, "maxTokens", 1000),
					"temperature": num(ctx.Params, "temperature", 0.7),
				}
				result, err := openaiCall(apiKey, "https://api.openai.com/v1/chat/completions", body)
				if err != nil {
					return schema.NodeResult{}, err
				}
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {item(result)}}}, nil

			case "completion:create":
				prompt := str(ctx.Params, "prompt", "")
				if prompt == "" {
					return schema.NodeResult{}, fmt.Errorf("openai: prompt is required")
				}
				body := map[string]any{
					"model":       model,
					"prompt":      prompt,
					"max_tokens":  num(ctx.Params, "maxTokens", 1000),
					"temperature": num(ctx.Params, "temperature", 0.7),
				}
				result, err := openaiCall(apiKey, "https://api.openai.com/v1/completions", body)
				if err != nil {
					return schema.NodeResult{}, err
				}
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {item(result)}}}, nil

			case "embedding:create":
				prompt := str(ctx.Params, "prompt", "")
				if prompt == "" {
					return schema.NodeResult{}, fmt.Errorf("openai: prompt is required")
				}
				input := any(prompt)
				if arr, ok := ctx.Params["prompt"].([]any); ok {
					input = arr
				}
				body := map[string]any{"model": model, "input": input}
				result, err := openaiCall(apiKey, "https://api.openai.com/v1/embeddings", body)
				if err != nil {
					return schema.NodeResult{}, err
				}
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {item(result)}}}, nil

			case "image:generate":
				prompt := str(ctx.Params, "prompt", "")
				if prompt == "" {
					return schema.NodeResult{}, fmt.Errorf("openai: prompt is required")
				}
				size := str(ctx.Params, "imageSize", "1024x1024")
				body := map[string]any{
					"model":  model,
					"prompt": prompt,
					"n":      1,
					"size":   size,
				}
				result, err := openaiCall(apiKey, "https://api.openai.com/v1/images/generations", body)
				if err != nil {
					return schema.NodeResult{}, err
				}
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {item(result)}}}, nil

			case "audio:transcribe", "audio:translate":
				audioFile := str(ctx.Params, "audioFile", "")
				if audioFile == "" {
					return schema.NodeResult{}, fmt.Errorf("openai: audioFile is required")
				}
				// For simplicity, handle URL or base64 by sending as multipart reference
				body := map[string]any{
					"model": model,
					"file":  audioFile,
				}
				if lang := str(ctx.Params, "language", ""); lang != "" {
					body["language"] = lang
				}
				endpoint := "https://api.openai.com/v1/audio/transcriptions"
				if op == "audio:translate" {
					endpoint = "https://api.openai.com/v1/audio/translations"
				}
				result, err := openaiCall(apiKey, endpoint, body)
				if err != nil {
					return schema.NodeResult{}, err
				}
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {item(result)}}}, nil

			case "moderation:check":
				prompt := str(ctx.Params, "prompt", "")
				if prompt == "" {
					return schema.NodeResult{}, fmt.Errorf("openai: prompt is required")
				}
				body := map[string]any{"input": prompt}
				result, err := openaiCall(apiKey, "https://api.openai.com/v1/moderations", body)
				if err != nil {
					return schema.NodeResult{}, err
				}
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {item(result)}}}, nil

			case "model:list":
				result, err := openaiCall(apiKey, "https://api.openai.com/v1/models", nil, http.MethodGet)
				if err != nil {
					return schema.NodeResult{}, err
				}
				items := toItems(result, "data")
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": items}}, nil

			default:
				return schema.NodeResult{}, fmt.Errorf("openai: unknown operation %q", op)
			}
		},
	}
}

func openaiCall(apiKey, url string, body map[string]any, method ...string) (map[string]any, error) {
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
		return nil, fmt.Errorf("openai: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := aiClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("openai: %d — %s", resp.StatusCode, trunc(string(raw), 400))
	}
	var result map[string]any
	if json.Unmarshal(raw, &result) != nil {
		return map[string]any{"raw": string(raw)}, nil
	}
	return result, nil
}
