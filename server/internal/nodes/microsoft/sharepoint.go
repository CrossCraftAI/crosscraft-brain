package microsoft

import (
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/rest"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// SharePoint — sites, lists, list items, and document libraries via Graph.
func SharePoint(base string) rest.Node {
	siteID := sp("siteId", "Site ID", true)
	listID := sp("listId", "List ID", true)
	itemID := sp("itemId", "Item ID", true)
	driveID := sp("driveId", "Drive ID", true)
	body := jp("body", "Body (JSON)")
	top := schema.ParamSchema{Name: "$top", Label: "Max results", Type: "number", Default: float64(50)}
	filter := schema.ParamSchema{Name: "$filter", Label: "Filter (OData)", Type: "string"}
	searchQ := schema.ParamSchema{Name: "q", Label: "Search query", Type: "string"}
	listName := schema.ParamSchema{Name: "displayName", Label: "Display name", Type: "string"}
	listDesc := schema.ParamSchema{Name: "description", Label: "Description", Type: "string"}
	selectParam := schema.ParamSchema{Name: "$select", Label: "Select fields", Type: "string", Placeholder: "id,title,createdDateTime"}
	expand := schema.ParamSchema{Name: "$expand", Label: "Expand relations", Type: "string"}

	return node(base, "microsoft.sharepoint", "SharePoint", "Globe", "SharePoint sites, lists, items, and document libraries.", []rest.Op{
		// Sites
		{Resource: "site", Name: "get", Label: "Get Site", Method: "GET", Path: "/sites/{siteId}", Params: []schema.ParamSchema{siteID}},
		{Resource: "site", Name: "getByPath", Label: "Get by Path", Method: "GET", Path: "/sites/{siteId}:/{sitePath}", Params: []schema.ParamSchema{siteID, sp("sitePath", "Site path", true)}},
		{Resource: "site", Name: "listRoot", Label: "Root Site", Method: "GET", Path: "/sites/root"},
		{Resource: "site", Name: "search", Label: "Search Sites", Method: "GET", Path: "/sites",
			Query: map[string]string{"search": "q"}, ItemsPath: "value", Params: []schema.ParamSchema{searchQ}},
		{Resource: "site", Name: "listAll", Label: "List All Sites", Method: "GET", Path: "/sites",
			Query: map[string]string{"$top": "$top"}, ItemsPath: "value", Params: []schema.ParamSchema{top}},
		// Lists
		{Resource: "list", Name: "list", Label: "List Lists", Method: "GET", Path: "/sites/{siteId}/lists", ItemsPath: "value",
			Params: []schema.ParamSchema{siteID, top, filter}},
		{Resource: "list", Name: "get", Label: "Get List", Method: "GET", Path: "/sites/{siteId}/lists/{listId}",
			Params: []schema.ParamSchema{siteID, listID, selectParam, expand}},
		{Resource: "list", Name: "create", Label: "Create List", Method: "POST", Path: "/sites/{siteId}/lists", BodyParam: "body",
			Params: []schema.ParamSchema{siteID, listName, listDesc, body}},
		{Resource: "list", Name: "update", Label: "Update List", Method: "PATCH", Path: "/sites/{siteId}/lists/{listId}", BodyParam: "body",
			Params: []schema.ParamSchema{siteID, listID, listName, body}},
		{Resource: "list", Name: "delete", Label: "Delete List", Method: "DELETE", Path: "/sites/{siteId}/lists/{listId}",
			Params: []schema.ParamSchema{siteID, listID}},
		// List items
		{Resource: "listItem", Name: "list", Label: "List Items", Method: "GET", Path: "/sites/{siteId}/lists/{listId}/items", ItemsPath: "value",
			Query: map[string]string{"$top": "$top", "$filter": "$filter", "$select": "$select", "$expand": "$expand"},
			Params: []schema.ParamSchema{siteID, listID, top, filter, selectParam, expand}},
		{Resource: "listItem", Name: "get", Label: "Get Item", Method: "GET", Path: "/sites/{siteId}/lists/{listId}/items/{itemId}",
			Params: []schema.ParamSchema{siteID, listID, itemID, selectParam, expand}},
		{Resource: "listItem", Name: "create", Label: "Create Item", Method: "POST", Path: "/sites/{siteId}/lists/{listId}/items", BodyParam: "body",
			Params: []schema.ParamSchema{siteID, listID, body}},
		{Resource: "listItem", Name: "update", Label: "Update Item", Method: "PATCH", Path: "/sites/{siteId}/lists/{listId}/items/{itemId}", BodyParam: "body",
			Params: []schema.ParamSchema{siteID, listID, itemID, body}},
		{Resource: "listItem", Name: "delete", Label: "Delete Item", Method: "DELETE", Path: "/sites/{siteId}/lists/{listId}/items/{itemId}",
			Params: []schema.ParamSchema{siteID, listID, itemID}},
		// Drives
		{Resource: "drive", Name: "list", Label: "List Drives", Method: "GET", Path: "/sites/{siteId}/drives", ItemsPath: "value",
			Params: []schema.ParamSchema{siteID, top}},
		{Resource: "drive", Name: "get", Label: "Get Drive", Method: "GET", Path: "/sites/{siteId}/drives/{driveId}",
			Params: []schema.ParamSchema{siteID, driveID}},
		{Resource: "drive", Name: "listRoot", Label: "List Drive Root", Method: "GET", Path: "/sites/{siteId}/drives/{driveId}/root/children", ItemsPath: "value",
			Params: []schema.ParamSchema{siteID, driveID, top}},
	})
}
