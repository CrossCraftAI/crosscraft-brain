package marketing

import (
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/rest"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// ActiveCampaign — email marketing, marketing automation, and CRM.
// Auth: Api-Token header via activecampaignApi credential.
// The base URL depends on your account (e.g. https://youraccount.api-us1.com).
func ActiveCampaign() rest.Node {
	contactID := sp("contactId", "Contact ID", true)
	automationID := sp("automationId", "Automation ID", true)
	campaignID := sp("campaignId", "Campaign ID", true)
	body := jp("body", "Body (JSON)")
	limit := ip("limit", "Max results", 100)

	return rest.Node{
		Type: "marketing.activecampaign", Label: "ActiveCampaign", Group: "integration", Icon: "MailPlus",
		Description:  "Manage ActiveCampaign contacts, lists, automations, and campaigns.",
		BaseURL:      "https://youraccount.api-us1.com/api/3",
		BaseURLParam: "baseUrl",
		CredType:     "activecampaignApi",
		Auth:         rest.Auth{Kind: "header", Header: "Api-Token", ValueField: "apiKey"},
		Ops: []rest.Op{
			// Contacts
			{Resource: "contact", Name: "list", Label: "List Contacts", Method: "GET",
				Path: "/contacts", ItemsPath: "contacts",
				Query:  map[string]string{"limit": "limit"},
				Params: []schema.ParamSchema{limit}},
			{Resource: "contact", Name: "create", Label: "Create Contact", Method: "POST",
				Path: "/contacts", BodyParam: "body", ItemsPath: "contact",
				Params: []schema.ParamSchema{body}},
			{Resource: "contact", Name: "update", Label: "Update Contact", Method: "PUT",
				Path: "/contacts/{contactId}", BodyParam: "body", ItemsPath: "contact",
				Params: []schema.ParamSchema{contactID, body}},
			{Resource: "contact", Name: "delete", Label: "Delete Contact", Method: "DELETE",
				Path: "/contacts/{contactId}",
				Params: []schema.ParamSchema{contactID}},
			// Lists
			{Resource: "list", Name: "list", Label: "List All Lists", Method: "GET",
				Path: "/lists", ItemsPath: "lists",
				Query:  map[string]string{"limit": "limit"},
				Params: []schema.ParamSchema{limit}},
			{Resource: "list", Name: "create", Label: "Create List", Method: "POST",
				Path: "/lists", BodyParam: "body", ItemsPath: "list",
				Params: []schema.ParamSchema{body}},
			// Automations
			{Resource: "automation", Name: "list", Label: "List Automations", Method: "GET",
				Path: "/automations", ItemsPath: "automations",
				Query:  map[string]string{"limit": "limit"},
				Params: []schema.ParamSchema{limit}},
			{Resource: "automation", Name: "trigger", Label: "Trigger Automation", Method: "POST",
				Path: "/automations/{automationId}/triggers", BodyParam: "body",
				Params: []schema.ParamSchema{automationID, body}},
			// Campaigns
			{Resource: "campaign", Name: "list", Label: "List Campaigns", Method: "GET",
				Path: "/campaigns", ItemsPath: "campaigns",
				Query:  map[string]string{"limit": "limit"},
				Params: []schema.ParamSchema{limit}},
			{Resource: "campaign", Name: "create", Label: "Create Campaign", Method: "POST",
				Path: "/campaigns", BodyParam: "body", ItemsPath: "campaign",
				Params: []schema.ParamSchema{body}},
			// Messages
			{Resource: "message", Name: "send", Label: "Send Campaign", Method: "POST",
				Path: "/campaigns/{campaignId}/send",
				Params: []schema.ParamSchema{campaignID}},
		},
	}
}
