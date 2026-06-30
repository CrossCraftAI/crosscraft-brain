package adobe

import (
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/rest"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// Firefly — Adobe Firefly generative AI API (text-to-image, generative fill,
// generative expand) via server-to-server IMS OAuth2.
func Firefly(base string) rest.Node {
	body := jp("body", "Body (JSON)")
	outputProp := schema.ParamSchema{Name: "outputProperty", Label: "Output binary property", Type: "string", Default: "data"}

	return oauth2Node(base,
		"adobe.firefly", "Adobe Firefly", "Sparkles",
		"Generate images from text prompts, fill and expand with generative AI.",
		[]rest.Op{
			{Resource: "image", Name: "generate", Label: "Generate Image (text-to-image)", Method: "POST",
				Path: "/v3/images/generate", BodyParam: "body",
				Params: []schema.ParamSchema{body, outputProp}},
			{Resource: "image", Name: "fill", Label: "Generative Fill", Method: "POST",
				Path: "/v3/images/fill", BodyParam: "body",
				Params: []schema.ParamSchema{body, outputProp}},
			{Resource: "image", Name: "expand", Label: "Generative Expand", Method: "POST",
				Path: "/v3/images/expand", BodyParam: "body",
				Params: []schema.ParamSchema{body, outputProp}},
			{Resource: "image", Name: "upscale", Label: "Generative Upscale", Method: "POST",
				Path: "/v3/images/upscale", BodyParam: "body",
				Params: []schema.ParamSchema{body, outputProp}},
		})
}
