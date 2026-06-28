package microsoft

import (
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/rest"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// Outlook — mail via Graph. Enhanced with reply, move, drafts, attachments,
// folder CRUD, and message/attachment operations.
func Outlook(base string) rest.Node {
	msgID := sp("messageId", "Message ID", true)
	body := jp("body", "Body (JSON)")
	top := schema.ParamSchema{Name: "$top", Label: "Max results", Type: "number", Default: float64(25)}
	filter := schema.ParamSchema{Name: "$filter", Label: "Filter (OData)", Type: "string"}
	toParam := schema.ParamSchema{Name: "to", Label: "To", Type: "string", Placeholder: "user@example.com"}
	subject := schema.ParamSchema{Name: "subject", Label: "Subject", Type: "string"}
	contentType := schema.ParamSchema{Name: "contentType", Label: "Content type", Type: "select", Default: "Text",
		Options: []schema.ParamOption{{Label: "Text", Value: "Text"}, {Label: "HTML", Value: "HTML"}}}
	content := schema.ParamSchema{Name: "content", Label: "Content", Type: "code"}
	ccParam := schema.ParamSchema{Name: "cc", Label: "CC", Type: "string"}
	attachmentsParam := jp("attachments", "Attachments (JSON)")
	folderID := sp("folderId", "Folder ID", true)
	folderName := schema.ParamSchema{Name: "name", Label: "Folder name", Type: "string"}
	queryParam := schema.ParamSchema{Name: "q", Label: "Search query", Type: "string", Placeholder: "subject:urgent"}
	attachmentID := sp("attachmentId", "Attachment ID", true)
	attachmentName := schema.ParamSchema{Name: "attachmentName", Label: "Attachment Name", Type: "string"}
	saveToSent := schema.ParamSchema{Name: "saveToSentItems", Label: "Save to sent", Type: "boolean", Default: true}

	return node(base, "microsoft.outlook", "Outlook", "Mail", "Read, send, reply, and manage Outlook mail, drafts, folders, and attachments.", []rest.Op{
		// Messages
		{Resource: "message", Name: "list", Label: "List Messages", Method: "GET", Path: "/me/messages", ItemsPath: "value",
			Query: map[string]string{"$top": "$top", "$filter": "$filter", "$search": "q"}, Params: []schema.ParamSchema{top, filter, queryParam}},
		{Resource: "message", Name: "get", Label: "Get Message", Method: "GET", Path: "/me/messages/{messageId}", Params: []schema.ParamSchema{msgID}},
		{Resource: "message", Name: "send", Label: "Send Mail", Method: "POST", Path: "/me/sendMail", BodyParam: "body",
			Params: []schema.ParamSchema{toParam, subject, content, contentType, ccParam, attachmentsParam, saveToSent, body}},
		{Resource: "message", Name: "delete", Label: "Delete Message", Method: "DELETE", Path: "/me/messages/{messageId}", Params: []schema.ParamSchema{msgID}},
		{Resource: "message", Name: "reply", Label: "Reply to Message", Method: "POST", Path: "/me/messages/{messageId}/reply", BodyParam: "body",
			Params: []schema.ParamSchema{msgID, content, contentType, body}},
		{Resource: "message", Name: "replyAll", Label: "Reply All", Method: "POST", Path: "/me/messages/{messageId}/replyAll", BodyParam: "body",
			Params: []schema.ParamSchema{msgID, content, contentType, body}},
		{Resource: "message", Name: "move", Label: "Move to Folder", Method: "POST", Path: "/me/messages/{messageId}/move", BodyParam: "body",
			Params: []schema.ParamSchema{msgID, folderID, body}},
		{Resource: "message", Name: "forward", Label: "Forward", Method: "POST", Path: "/me/messages/{messageId}/forward", BodyParam: "body",
			Params: []schema.ParamSchema{msgID, toParam, content, contentType, body}},
		// Drafts
		{Resource: "draft", Name: "list", Label: "List Drafts", Method: "GET", Path: "/me/messages",
			Query: map[string]string{"$top": "$top", "$filter": "=isDraft eq true"}, ItemsPath: "value", Params: []schema.ParamSchema{top}},
		{Resource: "draft", Name: "create", Label: "Create Draft", Method: "POST", Path: "/me/messages", BodyParam: "body",
			Params: []schema.ParamSchema{toParam, subject, content, contentType, ccParam, attachmentsParam, body}},
		{Resource: "draft", Name: "update", Label: "Update Draft", Method: "PATCH", Path: "/me/messages/{messageId}", BodyParam: "body",
			Params: []schema.ParamSchema{msgID, body}},
		{Resource: "draft", Name: "delete", Label: "Delete Draft", Method: "DELETE", Path: "/me/messages/{messageId}", Params: []schema.ParamSchema{msgID}},
		// Attachments
		{Resource: "attachment", Name: "list", Label: "List Attachments", Method: "GET", Path: "/me/messages/{messageId}/attachments", ItemsPath: "value",
			Params: []schema.ParamSchema{msgID}},
		{Resource: "attachment", Name: "get", Label: "Get Attachment", Method: "GET", Path: "/me/messages/{messageId}/attachments/{attachmentId}",
			Params: []schema.ParamSchema{msgID, attachmentID}},
		{Resource: "attachment", Name: "add", Label: "Add Attachment", Method: "POST", Path: "/me/messages/{messageId}/attachments", BodyParam: "body",
			Params: []schema.ParamSchema{msgID, attachmentName, body}},
		{Resource: "attachment", Name: "delete", Label: "Delete Attachment", Method: "DELETE", Path: "/me/messages/{messageId}/attachments/{attachmentId}",
			Params: []schema.ParamSchema{msgID, attachmentID}},
		// Folders
		{Resource: "folder", Name: "list", Label: "List Mail Folders", Method: "GET", Path: "/me/mailFolders", ItemsPath: "value",
			Params: []schema.ParamSchema{top}},
		{Resource: "folder", Name: "listChildren", Label: "List Child Folders", Method: "GET", Path: "/me/mailFolders/{folderId}/childFolders", ItemsPath: "value",
			Params: []schema.ParamSchema{folderID}},
		{Resource: "folder", Name: "get", Label: "Get Folder", Method: "GET", Path: "/me/mailFolders/{folderId}", Params: []schema.ParamSchema{folderID}},
		{Resource: "folder", Name: "create", Label: "Create Folder", Method: "POST", Path: "/me/mailFolders", BodyParam: "body",
			Params: []schema.ParamSchema{folderName}},
		{Resource: "folder", Name: "update", Label: "Update Folder", Method: "PATCH", Path: "/me/mailFolders/{folderId}", BodyParam: "body",
			Params: []schema.ParamSchema{folderID, folderName}},
		{Resource: "folder", Name: "delete", Label: "Delete Folder", Method: "DELETE", Path: "/me/mailFolders/{folderId}", Params: []schema.ParamSchema{folderID}},
	})
}
