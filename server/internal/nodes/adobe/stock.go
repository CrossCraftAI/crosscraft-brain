package adobe

import (
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/rest"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// Stock — Adobe Stock API (search, get details, license assets)
// via server-to-server IMS OAuth2.
func Stock(base string) rest.Node {
	body := jp("body", "Body (JSON)")
	assetID := sp("assetId", "Asset ID", true)
	licenseID := sp("licenseId", "License ID", true)
	binaryProp := schema.ParamSchema{Name: "binaryProperty", Label: "Binary property", Type: "string", Default: "data"}

	return oauth2Node(base,
		"adobe.stock", "Adobe Stock", "Image",
		"Search, get details, and license Adobe Stock assets.",
		[]rest.Op{
			{Resource: "asset", Name: "search", Label: "Search Assets", Method: "GET",
				Path: "/Rest/Media/1/Search/Files",
				Query: map[string]string{"locale": "=en_US"},
				Params: []schema.ParamSchema{body}},
			{Resource: "asset", Name: "get", Label: "Get Asset Details", Method: "GET",
				Path: "/Rest/Media/1/Files/{assetId}",
				Params: []schema.ParamSchema{assetID}},
			{Resource: "asset", Name: "license", Label: "License Asset", Method: "POST",
				Path: "/Rest/Libraries/1/Content/License", BodyParam: "body",
				Params: []schema.ParamSchema{assetID, body}},
			{Resource: "asset", Name: "download", Label: "Download Asset", Method: "GET",
				Path: "/Rest/Libraries/1/Content/Download/{licenseId}",
				Params: []schema.ParamSchema{licenseID, binaryProp}},
			{Resource: "license", Name: "list", Label: "List Licenses", Method: "GET",
				Path: "/Rest/Libraries/1/Content/License"},
			{Resource: "license", Name: "get", Label: "Get License Details", Method: "GET",
				Path: "/Rest/Libraries/1/Content/License/{licenseId}",
				Params: []schema.ParamSchema{licenseID}},
			{Resource: "collection", Name: "list", Label: "List Collections", Method: "GET",
				Path: "/Rest/Libraries/1/Collections"},
		})
}
