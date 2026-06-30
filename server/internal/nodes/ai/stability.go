package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// StabilityAI — Stability AI image generation (Stable Diffusion) and model listing.
func StabilityAI() schema.NodeDefinition {
	return schema.NodeDefinition{
		Type:        "ai.stability",
		Label:       "Stability AI",
		Group:       "ai",
		Icon:        "Image",
		Description: "Generate images from text prompts using Stability AI (Stable Diffusion).",
		Inputs:      []schema.Port{{ID: "main"}},
		Outputs:     []schema.Port{{ID: "main"}, {ID: "images", Label: "Images"}},
		Credentials: []string{"stabilityApi"},
		Params: []schema.ParamSchema{
			{Name: "credential", Label: "Credential", Type: "credential", Required: true, CredentialType: "stabilityApi"},
			{Name: "operation", Label: "Operation", Type: "select", Required: true, Options: []schema.ParamOption{
				{Label: "Generate Image", Value: "image:generate"},
				{Label: "Image Variation", Value: "image:variation"},
				{Label: "List Models", Value: "model:list"},
			}},
			{Name: "prompt", Label: "Prompt", Type: "expression", Required: true,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"image:generate", "image:variation"}}},
			{Name: "engineId", Label: "Engine / Model ID", Type: "string", Default: "stable-diffusion-xl-1024-v1-0",
				Placeholder: "stable-diffusion-xl-1024-v1-0"},
			{Name: "width", Label: "Width", Type: "number", Default: 1024,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"image:generate"}}},
			{Name: "height", Label: "Height", Type: "number", Default: 1024,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"image:generate"}}},
			{Name: "steps", Label: "Steps", Type: "number", Default: 30,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"image:generate"}}},
			{Name: "cfgScale", Label: "CFG Scale", Type: "number", Default: 7.0,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"image:generate"}}},
			{Name: "samples", Label: "Samples", Type: "number", Default: 1,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"image:generate"}}},
		},
		Execute: func(ctx *schema.ExecContext) (schema.NodeResult, error) {
			op := str(ctx.Params, "operation", "image:generate")
			apiKey := getCredStr(ctx, "credential", "apiKey", "")
			engineID := str(ctx.Params, "engineId", "stable-diffusion-xl-1024-v1-0")

			switch op {
			case "image:generate":
				prompt := str(ctx.Params, "prompt", "")
				if prompt == "" {
					return schema.NodeResult{}, fmt.Errorf("stability: prompt is required")
				}
				body := map[string]any{
					"text_prompts": []map[string]any{{"text": prompt, "weight": 1.0}},
					"cfg_scale":    num(ctx.Params, "cfgScale", 7.0),
					"steps":        num(ctx.Params, "steps", 30),
					"width":        num(ctx.Params, "width", 1024),
					"height":       num(ctx.Params, "height", 1024),
					"samples":      num(ctx.Params, "samples", 1),
				}
				result, err := stabilityCall(apiKey,
					"https://api.stability.ai/v1/generation/"+engineID+"/text-to-image", body)
				if err != nil {
					return schema.NodeResult{}, err
				}
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {item(result)}}}, nil

			case "image:variation":
				prompt := str(ctx.Params, "prompt", "")
				if prompt == "" {
					return schema.NodeResult{}, fmt.Errorf("stability: prompt is required")
				}
				body := map[string]any{
					"text_prompts": []map[string]any{{"text": prompt, "weight": 0.5}},
				}
				result, err := stabilityCall(apiKey,
					"https://api.stability.ai/v1/generation/"+engineID+"/image-to-image", body)
				if err != nil {
					return schema.NodeResult{}, err
				}
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {item(result)}}}, nil

			case "model:list":
				result, err := stabilityCall(apiKey, "https://api.stability.ai/v1/engines/list", nil)
				if err != nil {
					return schema.NodeResult{}, err
				}
				items := toItems(result, "data")
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": items}}, nil

			default:
				return schema.NodeResult{}, fmt.Errorf("stability: unknown operation %q", op)
			}
		},
	}
}

func stabilityCall(apiKey, url string, body map[string]any) (map[string]any, error) {
	var bodyBytes []byte
	if body != nil {
		bodyBytes, _ = json.Marshal(body)
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("stability: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := aiClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("stability: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("stability: %d — %s", resp.StatusCode, trunc(string(raw), 400))
	}
	var result map[string]any
	if json.Unmarshal(raw, &result) != nil {
		return map[string]any{"raw": string(raw)}, nil
	}
	return result, nil
}
