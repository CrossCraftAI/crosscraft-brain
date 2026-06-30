package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

var aiClient = &http.Client{Timeout: 60 * time.Second}

// HF helper funcs for this file
func hfJSON(v any) map[string]any { m, _ := v.(map[string]any); return m }

// HuggingFace — Hugging Face Inference API node. Runs inference on hosted models
// for text generation, translation, summarisation, classification, and embeddings.
func HuggingFace() schema.NodeDefinition {
	return schema.NodeDefinition{
		Type:        "ai.huggingface",
		Label:       "Hugging Face",
		Group:       "ai",
		Icon:        "BrainCircuit",
		Description: "Run inference on Hugging Face models: text generation, embeddings, classification, and more.",
		Inputs:      []schema.Port{{ID: "main"}},
		Outputs:     []schema.Port{{ID: "main"}},
		Credentials: []string{"huggingfaceApi"},
		Params: []schema.ParamSchema{
			{Name: "credential", Label: "Credential", Type: "credential", Required: true, CredentialType: "huggingfaceApi"},
			{Name: "operation", Label: "Operation", Type: "select", Required: true, Options: []schema.ParamOption{
				{Label: "Run Inference", Value: "inference:run"},
				{Label: "List Models", Value: "model:list"},
				{Label: "Get Model", Value: "model:get"},
				{Label: "Create Embedding", Value: "embedding:create"},
			}},
			{Name: "modelId", Label: "Model ID", Type: "string", Required: true,
				Placeholder: "gpt2 / sentence-transformers/all-MiniLM-L6-v2"},
			{Name: "input", Label: "Input Text", Type: "expression", Required: true,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"inference:run", "embedding:create"}}},
			{Name: "parameters", Label: "Parameters (JSON)", Type: "json",
				Description: "Model-specific parameters (e.g. {max_length:100, temperature:0.7})."},
		},
		Execute: func(ctx *schema.ExecContext) (schema.NodeResult, error) {
			op := str(ctx.Params, "operation", "inference:run")
			modelID := str(ctx.Params, "modelId", "")
			if modelID == "" {
				return schema.NodeResult{}, fmt.Errorf("huggingface: modelId is required")
			}
			apiKey := getCredStr(ctx, "credential", "apiKey", "")

			switch op {
			case "inference:run":
				input := str(ctx.Params, "input", "")
				params := rawParam(ctx.Params, "parameters")
				body := map[string]any{"inputs": input}
				if params != nil {
					body["parameters"] = params
				}
				result, err := hfCall(apiKey, http.MethodPost,
					"https://api-inference.huggingface.co/models/"+modelID, body)
				if err != nil {
					return schema.NodeResult{}, err
				}
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {item(result)}}}, nil

			case "model:list":
				result, err := hfCall(apiKey, http.MethodGet,
					"https://huggingface.co/api/models?limit=20", nil)
				if err != nil {
					return schema.NodeResult{}, err
				}
				items := toItems(result, "")
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": items}}, nil

			case "model:get":
				result, err := hfCall(apiKey, http.MethodGet,
					"https://huggingface.co/api/models/"+modelID, nil)
				if err != nil {
					return schema.NodeResult{}, err
				}
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {item(result)}}}, nil

			case "embedding:create":
				input := str(ctx.Params, "input", "")
				result, err := hfCall(apiKey, http.MethodPost,
					"https://api-inference.huggingface.co/models/"+modelID, map[string]any{"inputs": input})
				if err != nil {
					return schema.NodeResult{}, err
				}
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {item(result)}}}, nil

			default:
				return schema.NodeResult{}, fmt.Errorf("huggingface: unknown operation %q", op)
			}
		},
	}
}

func hfCall(apiKey, method, url string, body map[string]any) (map[string]any, error) {
	var bodyBytes []byte
	if body != nil {
		bodyBytes, _ = json.Marshal(body)
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("huggingface: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := aiClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("huggingface: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("huggingface: %d — %s", resp.StatusCode, trunc(string(raw), 400))
	}
	var result any
	if json.Unmarshal(raw, &result) != nil {
		return map[string]any{"raw": string(raw)}, nil
	}
	if m, ok := result.(map[string]any); ok {
		return m, nil
	}
	if arr, ok := result.([]any); ok {
		return map[string]any{"results": arr}, nil
	}
	return map[string]any{"value": result}, nil
}
