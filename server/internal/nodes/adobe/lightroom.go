package adobe

import (
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/rest"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// Lightroom — Adobe Lightroom API (auto-tone, apply presets, edit images)
// via server-to-server IMS OAuth2.
func Lightroom(base string) rest.Node {
	body := jp("body", "Body (JSON)")
	binaryProp := schema.ParamSchema{Name: "binaryProperty", Label: "Binary property", Type: "string", Default: "data"}
	outputProp := schema.ParamSchema{Name: "outputProperty", Label: "Output binary property", Type: "string", Default: "data"}
	assetID := sp("assetId", "Asset ID (catalog asset)", true)
	catalogID := sp("catalogId", "Catalog ID", true)

	return oauth2Node(base,
		"adobe.lightroom", "Adobe Lightroom", "Sun",
		"Apply auto-tone, presets, and edits to photos.",
		[]rest.Op{
			{Resource: "image", Name: "autoTone", Label: "Auto Tone", Method: "PUT",
				Path: "/v2/catalogs/{catalogId}/assets/{assetId}/autoTone", BodyParam: "body",
				Params: []schema.ParamSchema{catalogID, assetID, body}},
			{Resource: "image", Name: "applyPreset", Label: "Apply Preset", Method: "PUT",
				Path: "/v2/catalogs/{catalogId}/assets/{assetId}/preset", BodyParam: "body",
				Params: []schema.ParamSchema{catalogID, assetID, body, outputProp}},
			{Resource: "image", Name: "edit", Label: "Edit Image", Method: "PUT",
				Path: "/v2/catalogs/{catalogId}/assets/{assetId}/edit", BodyParam: "body",
				Params: []schema.ParamSchema{catalogID, assetID, body, outputProp}},
			{Resource: "image", Name: "getRendition", Label: "Get Rendition", Method: "GET",
				Path: "/v2/catalogs/{catalogId}/assets/{assetId}/renditions",
				Params: []schema.ParamSchema{catalogID, assetID}},
			// Asset management
			{Resource: "asset", Name: "list", Label: "List Assets", Method: "GET",
				Path: "/v2/catalogs/{catalogId}/assets",
				Params: []schema.ParamSchema{catalogID}},
			{Resource: "asset", Name: "get", Label: "Get Asset", Method: "GET",
				Path: "/v2/catalogs/{catalogId}/assets/{assetId}",
				Params: []schema.ParamSchema{catalogID, assetID}},
			{Resource: "asset", Name: "upload", Label: "Upload Asset", Method: "POST",
				Path: "/v2/catalogs/{catalogId}/assets", BodyParam: "body",
				Params: []schema.ParamSchema{catalogID, binaryProp, body}},
		})
}
