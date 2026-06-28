package adobe

import (
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/rest"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// AEMAssets — Adobe Experience Manager Assets API (upload, get, update metadata,
// get renditions). Base URL is user-overridable since AEM is self-hosted or
// AEM-as-a-Cloud-Service. Auth via server-to-server IMS OAuth2.
func AEMAssets(base string) rest.Node {
	assetID := sp("assetId", "Asset path or ID", true)
	body := jp("body", "Body (JSON)")
	binaryProp := schema.ParamSchema{Name: "binaryProperty", Label: "Binary property", Type: "string", Default: "data"}
	outputProp := schema.ParamSchema{Name: "outputProperty", Label: "Output binary property", Type: "string", Default: "data"}

	return rest.Node{
		Type: "adobe.aemAssets", Label: "Adobe AEM Assets", Group: "integration", Icon: "FolderOpen",
		Description:  "Upload, get, update metadata, and retrieve renditions from AEM Assets.",
		BaseURL:      base,
		BaseURLParam: "baseUrl",
		CredType:     "adobeOAuth2Api",
		Auth:         rest.Auth{Kind: "oauth2"},
		Ops: []rest.Op{
			{Resource: "asset", Name: "upload", Label: "Upload Asset", Method: "POST",
				Path: "/api/assets", BodyParam: "body",
				Params: []schema.ParamSchema{binaryProp, body}},
			{Resource: "asset", Name: "get", Label: "Get Asset Metadata", Method: "GET",
				Path: "/api/assets/{assetId}",
				Params: []schema.ParamSchema{assetID}},
			{Resource: "asset", Name: "updateMetadata", Label: "Update Metadata", Method: "PUT",
				Path: "/api/assets/{assetId}", BodyParam: "body",
				Params: []schema.ParamSchema{assetID, body}},
			{Resource: "asset", Name: "delete", Label: "Delete Asset", Method: "DELETE",
				Path: "/api/assets/{assetId}",
				Params: []schema.ParamSchema{assetID}},
			{Resource: "asset", Name: "getRendition", Label: "Get Rendition", Method: "GET",
				Path: "/api/assets/{assetId}/renditions",
				Params: []schema.ParamSchema{assetID, outputProp}},
			{Resource: "asset", Name: "list", Label: "List Assets", Method: "GET",
				Path: "/api/assets"},
			{Resource: "folder", Name: "list", Label: "List Folders", Method: "GET",
				Path: "/api/folders"},
			{Resource: "folder", Name: "create", Label: "Create Folder", Method: "POST",
				Path: "/api/folders", BodyParam: "body",
				Params: []schema.ParamSchema{body}},
		},
	}
}
