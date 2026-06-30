package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// ElevenLabs — text-to-speech and voice management.
func ElevenLabs() schema.NodeDefinition {
	return schema.NodeDefinition{
		Type:        "ai.elevenlabs",
		Label:       "ElevenLabs",
		Group:       "ai",
		Icon:        "Volume2",
		Description: "Text-to-speech generation and voice management via ElevenLabs.",
		Inputs:      []schema.Port{{ID: "main"}},
		Outputs:     []schema.Port{{ID: "main"}, {ID: "audio", Label: "Audio"}},
		Credentials: []string{"elevenlabsApi"},
		Params: []schema.ParamSchema{
			{Name: "credential", Label: "Credential", Type: "credential", Required: true, CredentialType: "elevenlabsApi"},
			{Name: "operation", Label: "Operation", Type: "select", Required: true, Options: []schema.ParamOption{
				{Label: "Text-to-Speech", Value: "tts:generate"},
				{Label: "List Voices", Value: "voice:list"},
				{Label: "Get Voice", Value: "voice:get"},
			}},
			{Name: "voiceId", Label: "Voice ID", Type: "string",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"tts:generate", "voice:get"}},
				Placeholder: "21m00Tcm4TlvDq8ikWAM (Rachel)"},
			{Name: "text", Label: "Text", Type: "expression", Required: true,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"tts:generate"}}},
			{Name: "modelId", Label: "Model ID", Type: "string", Default: "eleven_multilingual_v2",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"tts:generate"}}},
			{Name: "stability", Label: "Stability", Type: "number", Default: 0.5,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"tts:generate"}}},
		},
		Execute: func(ctx *schema.ExecContext) (schema.NodeResult, error) {
			op := str(ctx.Params, "operation", "tts:generate")
			apiKey := getCredStr(ctx, "credential", "apiKey", "")

			switch op {
			case "tts:generate":
				voiceID := str(ctx.Params, "voiceId", "21m00Tcm4TlvDq8ikWAM")
				text := str(ctx.Params, "text", "")
				modelID := str(ctx.Params, "modelId", "eleven_multilingual_v2")
				if text == "" {
					return schema.NodeResult{}, fmt.Errorf("elevenlabs: text is required")
				}
				body := map[string]any{
					"text":     text,
					"model_id": modelID,
					"voice_settings": map[string]any{
						"stability":        num(ctx.Params, "stability", 0.5),
						"similarity_boost": 0.75,
					},
				}
				result, audioData, err := elevenlabsTTS(apiKey, voiceID, body)
				if err != nil {
					return schema.NodeResult{}, err
				}
				return schema.NodeResult{Outputs: map[string][]schema.Item{
					"main":  {item(result)},
					"audio": {schema.Item{Binary: map[string]schema.BinaryRef{"audio": {Data: audioData, MimeType: "audio/mpeg"}}}},
				}}, nil

			case "voice:list":
				result, err := elevenlabsCall(apiKey, http.MethodGet, "https://api.elevenlabs.io/v1/voices", nil)
				if err != nil {
					return schema.NodeResult{}, err
				}
				items := toItems(result, "voices")
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": items}}, nil

			case "voice:get":
				voiceID := str(ctx.Params, "voiceId", "")
				if voiceID == "" {
					return schema.NodeResult{}, fmt.Errorf("elevenlabs: voiceId is required")
				}
				result, err := elevenlabsCall(apiKey, http.MethodGet, "https://api.elevenlabs.io/v1/voices/"+voiceID, nil)
				if err != nil {
					return schema.NodeResult{}, err
				}
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {item(result)}}}, nil

			default:
				return schema.NodeResult{}, fmt.Errorf("elevenlabs: unknown operation %q", op)
			}
		},
	}
}

func elevenlabsTTS(apiKey, voiceID string, body map[string]any) (map[string]any, string, error) {
	bodyBytes, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost,
		"https://api.elevenlabs.io/v1/text-to-speech/"+voiceID, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, "", fmt.Errorf("elevenlabs: %w", err)
	}
	req.Header.Set("xi-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "audio/mpeg")
	resp, err := aiClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("elevenlabs: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, "", fmt.Errorf("elevenlabs: %d — %s", resp.StatusCode, trunc(string(raw), 400))
	}
	// Encode audio as base64 for JSON transport
	b64Audio := bytesToB64(raw)
	return map[string]any{
		"status":   "generated",
		"voiceId":  voiceID,
		"mimeType": "audio/mpeg",
		"size":     len(raw),
	}, b64Audio, nil
}

func elevenlabsCall(apiKey, method, url string, body map[string]any) (map[string]any, error) {
	var bodyBytes []byte
	if body != nil {
		bodyBytes, _ = json.Marshal(body)
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("elevenlabs: %w", err)
	}
	req.Header.Set("xi-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := aiClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("elevenlabs: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("elevenlabs: %d — %s", resp.StatusCode, trunc(string(raw), 400))
	}
	var result map[string]any
	if json.Unmarshal(raw, &result) != nil {
		return map[string]any{"raw": string(raw)}, nil
	}
	return result, nil
}
