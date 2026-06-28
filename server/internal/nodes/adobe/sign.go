package adobe

import (
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/rest"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// AcrobatSign — Adobe Acrobat Sign v6 (e-signature). Auth: integration key as a
// Bearer header (adobeSignApi). The base URL defaults to the na1 shard and is
// overridable via the per-node "baseUrl" param for other account regions.
func AcrobatSign(base string) rest.Node {
	agID := sp("agreementId", "Agreement ID", true)
	body := jp("body", "Body (JSON)")
	reminderMsg := schema.ParamSchema{Name: "message", Label: "Reminder message", Type: "string"}
	recipientEmail := schema.ParamSchema{Name: "email", Label: "Recipient email", Type: "string"}

	return rest.Node{
		Type: "adobe.acrobatSign", Label: "Adobe Acrobat Sign", Group: "integration", Icon: "FileSignature",
		Description:  "Send, track, and manage e-signature agreements with reminders and triggers.",
		BaseURL:      base,
		BaseURLParam: "baseUrl",
		CredType:     "adobeSignApi",
		Auth:         rest.Auth{Kind: "header", Header: "Authorization", Prefix: "Bearer ", ValueField: "accessToken"},
		Ops: []rest.Op{
			// Agreements
			{Resource: "agreement", Name: "list", Label: "List Agreements", Method: "GET",
				Path: "/agreements", ItemsPath: "userAgreementList"},
			{Resource: "agreement", Name: "get", Label: "Get Agreement", Method: "GET",
				Path: "/agreements/{agreementId}", Params: []schema.ParamSchema{agID}},
			{Resource: "agreement", Name: "create", Label: "Create Agreement", Method: "POST",
				Path: "/agreements", BodyParam: "body", Params: []schema.ParamSchema{body}},
			{Resource: "agreement", Name: "send", Label: "Send for Signature", Method: "POST",
				Path: "/agreements/{agreementId}/status", BodyParam: "body",
				Params: []schema.ParamSchema{agID, body}},
			{Resource: "agreement", Name: "cancel", Label: "Cancel Agreement", Method: "PUT",
				Path: "/agreements/{agreementId}/status", BodyParam: "body",
				Params: []schema.ParamSchema{agID, body}},
			{Resource: "agreement", Name: "getDocuments", Label: "Get Documents", Method: "GET",
				Path: "/agreements/{agreementId}/documents", ItemsPath: "documents",
				Params: []schema.ParamSchema{agID}},
			{Resource: "agreement", Name: "signingUrls", Label: "Get Signing URLs", Method: "GET",
				Path: "/agreements/{agreementId}/signingUrls", Params: []schema.ParamSchema{agID}},
			{Resource: "agreement", Name: "getEvents", Label: "Get Audit Events", Method: "GET",
				Path: "/agreements/{agreementId}/events", ItemsPath: "events",
				Params: []schema.ParamSchema{agID}},
			// Reminders
			{Resource: "reminder", Name: "send", Label: "Send Reminder", Method: "POST",
				Path: "/agreements/{agreementId}/reminders", BodyParam: "body",
				Params: []schema.ParamSchema{agID, recipientEmail, reminderMsg, body}},
			// Library Documents
			{Resource: "libraryDocument", Name: "list", Label: "List Library Documents", Method: "GET",
				Path: "/libraryDocuments", ItemsPath: "libraryDocumentList"},
			{Resource: "libraryDocument", Name: "get", Label: "Get Library Document", Method: "GET",
				Path: "/libraryDocuments/{libraryDocId}",
				Params: []schema.ParamSchema{sp("libraryDocId", "Library Document ID", true)}},
			// Webhooks
			{Resource: "webhook", Name: "list", Label: "List Webhooks", Method: "GET",
				Path: "/webhooks", ItemsPath: "webhookInfoList"},
			{Resource: "webhook", Name: "create", Label: "Create Webhook", Method: "POST",
				Path: "/webhooks", BodyParam: "body", Params: []schema.ParamSchema{body}},
		},
	}
}
