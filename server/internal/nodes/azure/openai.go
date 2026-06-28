package azure

import (
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/rest"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// OpenAI — Azure OpenAI Service REST API (completions, chat, embeddings, images,
// audio). Auth: API key via azureOpenAI credential.
func OpenAI(base string) rest.Node {
	body := jp("body", "Body (JSON)")
	deployment := sp("deployment", "Deployment name", true)
	apiVersion := schema.ParamSchema{Name: "apiVersion", Label: "API version", Type: "string", Default: "2024-02-15-preview"}

	return rest.Node{
		Type: "azure.openai", Label: "Azure OpenAI", Group: "integration", Icon: "Brain",
		Description:  "Generate completions, chat, embeddings, images, and audio transcriptions via Azure OpenAI.",
		BaseURL:      base,
		BaseURLParam: "baseUrl",
		CredType:     "azureOpenAI",
		Auth:         rest.Auth{Kind: "header", Header: "api-key", Prefix: "", ValueField: "accessToken"},
		Ops: []rest.Op{
			{Resource: "completion", Name: "create", Label: "Create Completion", Method: "POST",
				Path: "/openai/deployments/{deployment}/completions",
				Query: map[string]string{"api-version": "apiVersion"},
				BodyParam: "body",
				Params: []schema.ParamSchema{deployment, apiVersion, body}},
			{Resource: "chat", Name: "completion", Label: "Chat Completion", Method: "POST",
				Path: "/openai/deployments/{deployment}/chat/completions",
				Query: map[string]string{"api-version": "apiVersion"},
				BodyParam: "body",
				Params: []schema.ParamSchema{deployment, apiVersion, body}},
			{Resource: "embedding", Name: "create", Label: "Create Embeddings", Method: "POST",
				Path: "/openai/deployments/{deployment}/embeddings",
				Query: map[string]string{"api-version": "apiVersion"},
				BodyParam: "body",
				Params: []schema.ParamSchema{deployment, apiVersion, body}},
			{Resource: "image", Name: "generate", Label: "Generate Image (DALL-E)", Method: "POST",
				Path: "/openai/deployments/{deployment}/images/generations",
				Query: map[string]string{"api-version": "apiVersion"},
				BodyParam: "body",
				Params: []schema.ParamSchema{deployment, apiVersion, body}},
			{Resource: "audio", Name: "transcribe", Label: "Transcribe Audio (Whisper)", Method: "POST",
				Path: "/openai/deployments/{deployment}/audio/transcriptions",
				Query: map[string]string{"api-version": "apiVersion"},
				BodyParam: "body",
				Params: []schema.ParamSchema{deployment, apiVersion, body}},
			{Resource: "audio", Name: "translate", Label: "Translate Audio (Whisper)", Method: "POST",
				Path: "/openai/deployments/{deployment}/audio/translations",
				Query: map[string]string{"api-version": "apiVersion"},
				BodyParam: "body",
				Params: []schema.ParamSchema{deployment, apiVersion, body}},
		},
	}
}
