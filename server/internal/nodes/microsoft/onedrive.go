package microsoft

import (
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/rest"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// OneDrive — drive items via Graph. Enhanced with upload, download, copy, move,
// share, and search capabilities.
func OneDrive(base string) rest.Node {
	itemID := sp("itemId", "Item ID", true)
	body := jp("body", "Body (JSON)")
	filter := schema.ParamSchema{Name: "$filter", Label: "Filter (OData)", Type: "string"}
	top := schema.ParamSchema{Name: "$top", Label: "Max results", Type: "number", Default: float64(50)}
	fileName := schema.ParamSchema{Name: "name", Label: "Name", Type: "string"}
	parentID := sp("parentId", "Parent ID", true)
	targetID := sp("targetId", "Target folder ID", true)
	email := schema.ParamSchema{Name: "email", Label: "Email address", Type: "string"}
	role := schema.ParamSchema{Name: "role", Label: "Role", Type: "select", Default: "read",
		Options: []schema.ParamOption{{Label: "Viewer", Value: "read"}, {Label: "Editor", Value: "write"}}}
	binaryProp := schema.ParamSchema{Name: "binaryProperty", Label: "Binary property", Type: "string", Default: "data"}
	searchQ := schema.ParamSchema{Name: "q", Label: "Search query", Type: "string", Placeholder: "report.docx"}
	return node(base, "microsoft.onedrive", "OneDrive", "HardDrive", "List, manage, upload, download, copy, move, and share OneDrive files and folders.", []rest.Op{
		// List / search
		{Resource: "item", Name: "listRoot", Label: "List Root Children", Method: "GET", Path: "/me/drive/root/children", ItemsPath: "value",
			Params: []schema.ParamSchema{top, filter}},
		{Resource: "item", Name: "search", Label: "Search", Method: "GET", Path: "/me/drive/root/search(q='{q}')", ItemsPath: "value",
			Params: []schema.ParamSchema{searchQ, top}},
		{Resource: "item", Name: "recent", Label: "Recent Files", Method: "GET", Path: "/me/drive/recent", ItemsPath: "value",
			Params: []schema.ParamSchema{top}},
		{Resource: "item", Name: "sharedWithMe", Label: "Shared With Me", Method: "GET", Path: "/me/drive/sharedWithMe", ItemsPath: "value"},
		// Get / metadata
		{Resource: "item", Name: "get", Label: "Get Item", Method: "GET", Path: "/me/drive/items/{itemId}", Params: []schema.ParamSchema{itemID}},
		{Resource: "item", Name: "listChildren", Label: "List Children", Method: "GET", Path: "/me/drive/items/{itemId}/children", ItemsPath: "value",
			Params: []schema.ParamSchema{itemID, top}},
		// CRUD
		{Resource: "item", Name: "delete", Label: "Delete Item", Method: "DELETE", Path: "/me/drive/items/{itemId}", Params: []schema.ParamSchema{itemID}},
		{Resource: "folder", Name: "create", Label: "Create Folder", Method: "POST", Path: "/me/drive/root/children", BodyParam: "body",
			Params: []schema.ParamSchema{fileName, parentID, body}},
		{Resource: "folder", Name: "createInFolder", Label: "Create Folder In", Method: "POST", Path: "/me/drive/items/{parentId}/children", BodyParam: "body",
			Params: []schema.ParamSchema{parentID, fileName, body}},
		// Upload / download
		{Resource: "item", Name: "upload", Label: "Upload File", Method: "PUT", Path: "/me/drive/items/{parentId}:/{name}:/content", BodyParam: "body",
			Params: []schema.ParamSchema{parentID, fileName, binaryProp, body}},
		{Resource: "item", Name: "download", Label: "Download File", Method: "GET", Path: "/me/drive/items/{itemId}/content",
			Params: []schema.ParamSchema{itemID, binaryProp}},
		// Copy / move / share
		{Resource: "item", Name: "copy", Label: "Copy Item", Method: "POST", Path: "/me/drive/items/{itemId}/copy", BodyParam: "body",
			Params: []schema.ParamSchema{itemID, targetID, fileName, body}},
		{Resource: "item", Name: "move", Label: "Move Item", Method: "PATCH", Path: "/me/drive/items/{itemId}", BodyParam: "body",
			Params: []schema.ParamSchema{itemID, targetID, body}},
		{Resource: "item", Name: "share", Label: "Share Item", Method: "POST", Path: "/me/drive/items/{itemId}/invite", BodyParam: "body",
			Params: []schema.ParamSchema{itemID, email, role, body}},
		{Resource: "item", Name: "createLink", Label: "Create Share Link", Method: "POST", Path: "/me/drive/items/{itemId}/createLink", BodyParam: "body",
			Params: []schema.ParamSchema{itemID, role, body}},
		// Permissions
		{Resource: "permission", Name: "list", Label: "List Permissions", Method: "GET", Path: "/me/drive/items/{itemId}/permissions", ItemsPath: "value",
			Params: []schema.ParamSchema{itemID}},
		{Resource: "permission", Name: "delete", Label: "Delete Permission", Method: "DELETE", Path: "/me/drive/items/{itemId}/permissions/{permissionId}",
			Params: []schema.ParamSchema{itemID, sp("permissionId", "Permission ID", true)}},
	})
}
