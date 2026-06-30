package microsoft

import (
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/rest"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// Graph — raw authenticated Microsoft Graph API call. The escape hatch for
// any unwrapped API endpoint. Supports GET, POST, PATCH, PUT, and DELETE.
//
// The user provides the complete Graph API URL (including the version prefix)
// via the "rawUrl" parameter, which overrides the default Graph base URL.
// The path field on each op is empty so no path interpolation happens.
func Graph(base string) rest.Node {
	rawURL := schema.ParamSchema{Name: "rawUrl", Label: "Graph API URL", Type: "string",
		Placeholder: "https://graph.microsoft.com/v1.0/me/messages",
		Description: "Complete Graph API URL including version prefix. Overrides the default base URL."}
	body := jp("body", "Body (JSON)")

	return rest.Node{
		Type: "microsoft.graph", Label: "Microsoft Graph", Group: "integration", Icon: "Code",
		Description: "Raw authenticated Microsoft Graph API call — the escape hatch for any endpoint.",
		BaseURL: base, CredType: credType, Auth: rest.Auth{Kind: "oauth2"},
		BaseURLParam: "rawUrl",
		Ops: []rest.Op{
			{Resource: "raw", Name: "get", Label: "GET", Method: "GET", Path: "",
				Params: []schema.ParamSchema{rawURL}},
			{Resource: "raw", Name: "post", Label: "POST", Method: "POST", Path: "", BodyParam: "body",
				Params: []schema.ParamSchema{rawURL, body}},
			{Resource: "raw", Name: "patch", Label: "PATCH", Method: "PATCH", Path: "", BodyParam: "body",
				Params: []schema.ParamSchema{rawURL, body}},
			{Resource: "raw", Name: "put", Label: "PUT", Method: "PUT", Path: "", BodyParam: "body",
				Params: []schema.ParamSchema{rawURL, body}},
			{Resource: "raw", Name: "delete", Label: "DELETE", Method: "DELETE", Path: "",
				Params: []schema.ParamSchema{rawURL}},
		},
	}
}
