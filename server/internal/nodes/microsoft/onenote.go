package microsoft

import (
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/rest"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// OneNote — notebooks, sections, and pages via Graph.
func OneNote(base string) rest.Node {
	notebookID := sp("notebookId", "Notebook ID", true)
	sectionID := sp("sectionId", "Section ID", true)
	pageID := sp("pageId", "Page ID", true)
	body := jp("body", "Body (JSON)")
	top := schema.ParamSchema{Name: "$top", Label: "Max results", Type: "number", Default: float64(50)}
	filter := schema.ParamSchema{Name: "$filter", Label: "Filter (OData)", Type: "string"}
	pageName := schema.ParamSchema{Name: "title", Label: "Title", Type: "string"}
	contentType := schema.ParamSchema{Name: "contentType", Label: "Content type", Type: "select", Default: "Text",
		Options: []schema.ParamOption{{Label: "Text", Value: "Text"}, {Label: "HTML", Value: "Html"}}}

	return node(base, "microsoft.onenote", "OneNote", "BookOpen", "Manage OneNote notebooks, sections, and pages.", []rest.Op{
		// Notebooks
		{Resource: "notebook", Name: "list", Label: "List Notebooks", Method: "GET", Path: "/me/onenote/notebooks", ItemsPath: "value",
			Params: []schema.ParamSchema{top, filter}},
		{Resource: "notebook", Name: "get", Label: "Get Notebook", Method: "GET", Path: "/me/onenote/notebooks/{notebookId}",
			Params: []schema.ParamSchema{notebookID}},
		{Resource: "notebook", Name: "create", Label: "Create Notebook", Method: "POST", Path: "/me/onenote/notebooks", BodyParam: "body",
			Params: []schema.ParamSchema{pageName, body}},
		// Sections
		{Resource: "section", Name: "list", Label: "List Sections", Method: "GET", Path: "/me/onenote/notebooks/{notebookId}/sections", ItemsPath: "value",
			Params: []schema.ParamSchema{notebookID, top}},
		{Resource: "section", Name: "get", Label: "Get Section", Method: "GET", Path: "/me/onenote/sections/{sectionId}",
			Params: []schema.ParamSchema{sectionID}},
		{Resource: "section", Name: "create", Label: "Create Section", Method: "POST", Path: "/me/onenote/notebooks/{notebookId}/sections", BodyParam: "body",
			Params: []schema.ParamSchema{notebookID, pageName, body}},
		// Pages
		{Resource: "page", Name: "list", Label: "List Pages", Method: "GET", Path: "/me/onenote/sections/{sectionId}/pages", ItemsPath: "value",
			Params: []schema.ParamSchema{sectionID, top}},
		{Resource: "page", Name: "get", Label: "Get Page", Method: "GET", Path: "/me/onenote/pages/{pageId}",
			Query: map[string]string{"includeContent": "=true"}, Params: []schema.ParamSchema{pageID}},
		{Resource: "page", Name: "create", Label: "Create Page", Method: "POST", Path: "/me/onenote/sections/{sectionId}/pages", BodyParam: "body",
			Params: []schema.ParamSchema{sectionID, pageName, contentType, body}},
		{Resource: "page", Name: "delete", Label: "Delete Page", Method: "DELETE", Path: "/me/onenote/pages/{pageId}",
			Params: []schema.ParamSchema{pageID}},
	})
}
