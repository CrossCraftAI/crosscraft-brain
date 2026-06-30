package crm

import (
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/rest"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// ZohoCRM — Zoho CRM via REST API v7. Auth: OAuth2 Bearer token via zohoCrmApi
// credential. The base URL varies by data center (us, eu, in, au, jp, cn, ca);
// use the baseUrl param to override at runtime.
func ZohoCRM() rest.Node {
	leadID := sp("leadId", "Lead ID", true)
	contactID := sp("contactId", "Contact ID", true)
	accountID := sp("accountId", "Account ID", true)
	dealID := sp("dealId", "Deal ID", true)
	body := jp("body", "Body (JSON: {data:[{...}]})")
	limit := ip("limit", "Max results", 200)

	return rest.Node{
		Type: "crm.zoho", Label: "Zoho CRM", Group: "integration", Icon: "Building2",
		Description:  "Manage Zoho CRM leads, contacts, accounts, and deals.",
		BaseURL:      "https://www.zohoapis.com/crm/v7",
		BaseURLParam: "baseUrl",
		CredType:     "zohoCrmApi",
		Auth:         rest.Auth{Kind: "oauth2"},
		Ops: []rest.Op{
			// Leads
			{Resource: "lead", Name: "list", Label: "List Leads", Method: "GET",
				Path: "/Leads", ItemsPath: "data",
				Query:  map[string]string{"per_page": "limit"},
				Params: []schema.ParamSchema{limit}},
			{Resource: "lead", Name: "create", Label: "Create Lead", Method: "POST",
				Path: "/Leads", BodyParam: "body", ItemsPath: "data",
				Params: []schema.ParamSchema{body}},
			{Resource: "lead", Name: "update", Label: "Update Lead", Method: "PUT",
				Path: "/Leads/{leadId}", BodyParam: "body", ItemsPath: "data",
				Params: []schema.ParamSchema{leadID, body}},
			{Resource: "lead", Name: "delete", Label: "Delete Lead", Method: "DELETE",
				Path: "/Leads/{leadId}",
				Params: []schema.ParamSchema{leadID}},
			// Contacts
			{Resource: "contact", Name: "list", Label: "List Contacts", Method: "GET",
				Path: "/Contacts", ItemsPath: "data",
				Query:  map[string]string{"per_page": "limit"},
				Params: []schema.ParamSchema{limit}},
			{Resource: "contact", Name: "create", Label: "Create Contact", Method: "POST",
				Path: "/Contacts", BodyParam: "body", ItemsPath: "data",
				Params: []schema.ParamSchema{body}},
			{Resource: "contact", Name: "update", Label: "Update Contact", Method: "PUT",
				Path: "/Contacts/{contactId}", BodyParam: "body", ItemsPath: "data",
				Params: []schema.ParamSchema{contactID, body}},
			// Accounts
			{Resource: "account", Name: "list", Label: "List Accounts", Method: "GET",
				Path: "/Accounts", ItemsPath: "data",
				Query:  map[string]string{"per_page": "limit"},
				Params: []schema.ParamSchema{limit}},
			{Resource: "account", Name: "create", Label: "Create Account", Method: "POST",
				Path: "/Accounts", BodyParam: "body", ItemsPath: "data",
				Params: []schema.ParamSchema{body}},
			{Resource: "account", Name: "update", Label: "Update Account", Method: "PUT",
				Path: "/Accounts/{accountId}", BodyParam: "body", ItemsPath: "data",
				Params: []schema.ParamSchema{accountID, body}},
			// Deals
			{Resource: "deal", Name: "list", Label: "List Deals", Method: "GET",
				Path: "/Deals", ItemsPath: "data",
				Query:  map[string]string{"per_page": "limit"},
				Params: []schema.ParamSchema{limit}},
			{Resource: "deal", Name: "create", Label: "Create Deal", Method: "POST",
				Path: "/Deals", BodyParam: "body", ItemsPath: "data",
				Params: []schema.ParamSchema{body}},
			{Resource: "deal", Name: "update", Label: "Update Deal", Method: "PUT",
				Path: "/Deals/{dealId}", BodyParam: "body", ItemsPath: "data",
				Params: []schema.ParamSchema{dealID, body}},
		},
	}
}
