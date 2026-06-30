package marketing

import (
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/rest"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// Brevo (formerly Sendinblue) — email and SMS marketing platform.
// Auth: api-key header via brevoApi credential.
// API base: https://api.brevo.com/v3
func Brevo() rest.Node {
	contactID := sp("contactId", "Contact ID (email)", true)
	campaignID := sp("campaignId", "Campaign ID", true)
	templateID := sp("templateId", "Template ID", true)
	body := jp("body", "Body (JSON)")
	limit := ip("limit", "Max results", 50)

	return rest.Node{
		Type: "marketing.brevo", Label: "Brevo", Group: "integration", Icon: "Send",
		Description: "Manage Brevo contacts, email campaigns, and lists.",
		BaseURL:     "https://api.brevo.com/v3",
		CredType:    "brevoApi",
		Auth:        rest.Auth{Kind: "header", Header: "api-key", ValueField: "apiKey"},
		Ops: []rest.Op{
			// Contacts
			{Resource: "contact", Name: "list", Label: "List Contacts", Method: "GET",
				Path: "/contacts", ItemsPath: "contacts",
				Query:  map[string]string{"limit": "limit"},
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
			// Email
			{Resource: "email", Name: "send", Label: "Send Email", Method: "POST",
				Path: "/smtp/email", BodyParam: "body",
				Params: []schema.ParamSchema{body}},
			{Resource: "email", Name: "sendTemplate", Label: "Send Template Email", Method: "POST",
				Path: "/smtp/email", BodyParam: "body",
				Params: []schema.ParamSchema{templateID, body}},
			// Campaigns
			{Resource: "campaign", Name: "list", Label: "List Campaigns", Method: "GET",
				Path: "/emailCampaigns", ItemsPath: "campaigns",
				Query:  map[string]string{"limit": "limit"},
				Params: []schema.ParamSchema{limit}},
			{Resource: "campaign", Name: "create", Label: "Create Campaign", Method: "POST",
				Path: "/emailCampaigns", BodyParam: "body",
				Params: []schema.ParamSchema{body}},
			{Resource: "campaign", Name: "send", Label: "Send Campaign", Method: "POST",
				Path: "/emailCampaigns/{campaignId}/send",
				Params: []schema.ParamSchema{campaignID}},
			// Lists
			{Resource: "list", Name: "list", Label: "List All Lists", Method: "GET",
				Path: "/contacts/lists", ItemsPath: "lists",
				Query:  map[string]string{"limit": "limit"},
				Params: []schema.ParamSchema{limit}},
			{Resource: "list", Name: "create", Label: "Create List", Method: "POST",
				Path: "/contacts/lists", BodyParam: "body",
				Params: []schema.ParamSchema{body}},
		},
	}
}
