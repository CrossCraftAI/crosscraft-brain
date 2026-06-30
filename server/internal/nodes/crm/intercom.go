package crm

import (
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/rest"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// Intercom — Intercom customer messaging platform via REST API (v2.11).
// Auth: Bearer token via intercomApi credential.
func Intercom() rest.Node {
	contactID := sp("contactId", "Contact ID", true)
	conversationID := sp("conversationId", "Conversation ID", true)
	body := jp("body", "Body (JSON)")
	limit := ip("limit", "Max results", 150)

	return rest.Node{
		Type: "crm.intercom", Label: "Intercom", Group: "integration", Icon: "MessageCircle",
		Description: "Manage Intercom contacts, conversations, and companies.",
		BaseURL:     "https://api.intercom.io",
		CredType:    "intercomApi",
		Auth:        rest.Auth{Kind: "header", Header: "Authorization", Prefix: "Bearer ", ValueField: "accessToken"},
		Headers: map[string]string{
			"Intercom-Version": "2.11",
			"Accept":           "application/json",
		},
		Ops: []rest.Op{
			// Contacts
			{Resource: "contact", Name: "list", Label: "List Contacts", Method: "GET",
				Path: "/contacts", ItemsPath: "data",
				Query:  map[string]string{"per_page": "limit"},
				Params: []schema.ParamSchema{limit}},
			{Resource: "contact", Name: "create", Label: "Create Contact", Method: "POST",
				Path: "/contacts", BodyParam: "body",
				Params: []schema.ParamSchema{body}},
			{Resource: "contact", Name: "update", Label: "Update Contact", Method: "PUT",
				Path: "/contacts/{contactId}", BodyParam: "body",
				Params: []schema.ParamSchema{contactID, body}},
			{Resource: "contact", Name: "delete", Label: "Delete Contact", Method: "DELETE",
				Path: "/contacts/{contactId}",
				Params: []schema.ParamSchema{contactID}},
			// Conversations
			{Resource: "conversation", Name: "list", Label: "List Conversations", Method: "GET",
				Path: "/conversations", ItemsPath: "data",
				Query:  map[string]string{"per_page": "limit"},
				Params: []schema.ParamSchema{limit}},
			{Resource: "conversation", Name: "get", Label: "Get Conversation", Method: "GET",
				Path: "/conversations/{conversationId}",
				Params: []schema.ParamSchema{conversationID}},
			{Resource: "conversation", Name: "reply", Label: "Reply to Conversation", Method: "POST",
				Path: "/conversations/{conversationId}/reply", BodyParam: "body",
				Params: []schema.ParamSchema{conversationID, body}},
			{Resource: "conversation", Name: "close", Label: "Close Conversation", Method: "POST",
				Path: "/conversations/{conversationId}/parts", BodyParam: "body",
				Params: []schema.ParamSchema{conversationID, body}},
			// Companies
			{Resource: "company", Name: "list", Label: "List Companies", Method: "GET",
				Path: "/companies", ItemsPath: "data",
				Query:  map[string]string{"per_page": "limit"},
				Params: []schema.ParamSchema{limit}},
			{Resource: "company", Name: "create", Label: "Create Company", Method: "POST",
				Path: "/companies", BodyParam: "body",
				Params: []schema.ParamSchema{body}},
		},
	}
}
