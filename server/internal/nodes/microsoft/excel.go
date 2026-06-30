package microsoft

import (
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/rest"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// Excel — workbook worksheets, tables, ranges, and sessions via Graph.
// Enhanced with range CRUD, session management, and workbook creation.
func Excel(base string) rest.Node {
	itemID := sp("itemId", "Workbook (Drive item) ID", true)
	tableID := sp("tableId", "Table ID/Name", true)
	worksheetID := sp("worksheetId", "Worksheet ID/Name", true)
	body := jp("body", "Body (JSON)")
	address := schema.ParamSchema{Name: "address", Label: "Range address", Type: "string", Placeholder: "A1:C10", Default: "A1"}
	parentID := sp("parentId", "Parent folder ID", true)
	workbookName := schema.ParamSchema{Name: "name", Label: "Workbook name", Type: "string"}
	createEmpty := schema.ParamSchema{Name: "createEmpty", Label: "Create empty workbook", Type: "boolean", Default: false}

	return node(base, "microsoft.excel", "Microsoft Excel", "Table", "Workbook worksheets, tables, ranges, and sessions.", []rest.Op{
		// Worksheets
		{Resource: "worksheet", Name: "list", Label: "List Worksheets", Method: "GET", Path: "/me/drive/items/{itemId}/workbook/worksheets", ItemsPath: "value",
			Params: []schema.ParamSchema{itemID}},
		{Resource: "worksheet", Name: "get", Label: "Get Worksheet", Method: "GET", Path: "/me/drive/items/{itemId}/workbook/worksheets/{worksheetId}",
			Params: []schema.ParamSchema{itemID, worksheetID}},
		{Resource: "worksheet", Name: "create", Label: "Create Worksheet", Method: "POST", Path: "/me/drive/items/{itemId}/workbook/worksheets", BodyParam: "body",
			Params: []schema.ParamSchema{itemID, body}},
		{Resource: "worksheet", Name: "delete", Label: "Delete Worksheet", Method: "DELETE", Path: "/me/drive/items/{itemId}/workbook/worksheets/{worksheetId}",
			Params: []schema.ParamSchema{itemID, worksheetID}},
		{Resource: "worksheet", Name: "usedRange", Label: "Get Used Range", Method: "GET", Path: "/me/drive/items/{itemId}/workbook/worksheets/{worksheetId}/usedRange",
			Params: []schema.ParamSchema{itemID, worksheetID}},
		// Ranges
		{Resource: "range", Name: "get", Label: "Get Range", Method: "GET", Path: "/me/drive/items/{itemId}/workbook/worksheets/{worksheetId}/range(address='{address}')",
			Params: []schema.ParamSchema{itemID, worksheetID, address}},
		{Resource: "range", Name: "update", Label: "Update Range", Method: "PATCH", Path: "/me/drive/items/{itemId}/workbook/worksheets/{worksheetId}/range(address='{address}')", BodyParam: "body",
			Params: []schema.ParamSchema{itemID, worksheetID, address, body}},
		{Resource: "range", Name: "clear", Label: "Clear Range", Method: "POST", Path: "/me/drive/items/{itemId}/workbook/worksheets/{worksheetId}/range(address='{address}')/clear",
			Params: []schema.ParamSchema{itemID, worksheetID, address}},
		// Tables
		{Resource: "table", Name: "list", Label: "List Tables", Method: "GET", Path: "/me/drive/items/{itemId}/workbook/tables", ItemsPath: "value",
			Params: []schema.ParamSchema{itemID}},
		{Resource: "table", Name: "create", Label: "Create Table", Method: "POST", Path: "/me/drive/items/{itemId}/workbook/tables/add", BodyParam: "body",
			Params: []schema.ParamSchema{itemID, address, body}},
		{Resource: "table", Name: "addRow", Label: "Add Table Row", Method: "POST", Path: "/me/drive/items/{itemId}/workbook/tables/{tableId}/rows/add", BodyParam: "body",
			Params: []schema.ParamSchema{itemID, tableID, body}},
		{Resource: "table", Name: "getRows", Label: "List Table Rows", Method: "GET", Path: "/me/drive/items/{itemId}/workbook/tables/{tableId}/rows", ItemsPath: "value",
			Params: []schema.ParamSchema{itemID, tableID}},
		// Sessions
		{Resource: "session", Name: "create", Label: "Create Session", Method: "POST", Path: "/me/drive/items/{itemId}/workbook/createSession", BodyParam: "body",
			Params: []schema.ParamSchema{itemID}},
		{Resource: "session", Name: "refresh", Label: "Refresh Session", Method: "POST", Path: "/me/drive/items/{itemId}/workbook/refreshSession", BodyParam: "body",
			Params: []schema.ParamSchema{itemID}},
		{Resource: "session", Name: "close", Label: "Close Session", Method: "POST", Path: "/me/drive/items/{itemId}/workbook/closeSession",
			Params: []schema.ParamSchema{itemID}},
		// Workbook management (via OneDrive)
		{Resource: "workbook", Name: "create", Label: "Create Workbook", Method: "POST", Path: "/me/drive/items/{parentId}/children", BodyParam: "body",
			Params: []schema.ParamSchema{parentID, workbookName, createEmpty, body}},
	})
}
