package crm

import (
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/rest"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// Salesforce — Salesforce CRM via REST API (v58.0). Auth: OAuth2 Bearer token
// via salesforceApi credential. The base URL is overridable per instance.
func Salesforce(base string) rest.Node {
	body := jp("body", "Body (JSON)")
	query := schema.ParamSchema{Name: "query", Label: "SOQL Query", Type: "code"}
	limit := ip("limit", "Max results", 200)

	return rest.Node{
		Type: "crm.salesforce", Label: "Salesforce", Group: "integration", Icon: "Cloud",
		Description:  "Manage Salesforce accounts, contacts, leads, opportunities, and execute SOQL queries.",
		BaseURL:      base,
		BaseURLParam: "baseUrl",
		CredType:     "salesforceApi",
		Auth:         rest.Auth{Kind: "oauth2"},
		Ops: []rest.Op{
			// Accounts
			{Resource: "account", Name: "list", Label: "List Accounts", Method: "GET",
				Path: "/services/data/v58.0/query", ItemsPath: "records",
				Query:  map[string]string{"q": "=SELECT+Id,Name,Industry,Phone+FROM+Account+LIMIT+"},
				Params: []schema.ParamSchema{limit}},
			{Resource: "account", Name: "create", Label: "Create Account", Method: "POST",
				Path: "/services/data/v58.0/sobjects/Account", BodyParam: "body",
				Params: []schema.ParamSchema{body}},
			{Resource: "account", Name: "update", Label: "Update Account", Method: "PATCH",
				Path: "/services/data/v58.0/sobjects/Account/{accountId}", BodyParam: "body",
				Params: []schema.ParamSchema{sp("accountId", "Account ID", true), body}},
			{Resource: "account", Name: "delete", Label: "Delete Account", Method: "DELETE",
				Path: "/services/data/v58.0/sobjects/Account/{accountId}",
				Params: []schema.ParamSchema{sp("accountId", "Account ID", true)}},
			// Contacts
			{Resource: "contact", Name: "list", Label: "List Contacts", Method: "GET",
				Path: "/services/data/v58.0/query", ItemsPath: "records",
				Query:  map[string]string{"q": "=SELECT+Id,Name,Email+FROM+Contact+LIMIT+"},
				Params: []schema.ParamSchema{limit}},
			{Resource: "contact", Name: "create", Label: "Create Contact", Method: "POST",
				Path: "/services/data/v58.0/sobjects/Contact", BodyParam: "body",
				Params: []schema.ParamSchema{body}},
			{Resource: "contact", Name: "update", Label: "Update Contact", Method: "PATCH",
				Path: "/services/data/v58.0/sobjects/Contact/{contactId}", BodyParam: "body",
				Params: []schema.ParamSchema{sp("contactId", "Contact ID", true), body}},
			// Leads
			{Resource: "lead", Name: "list", Label: "List Leads", Method: "GET",
				Path: "/services/data/v58.0/query", ItemsPath: "records",
				Query:  map[string]string{"q": "=SELECT+Id,Name,Company,Status+FROM+Lead+LIMIT+"},
				Params: []schema.ParamSchema{limit}},
			{Resource: "lead", Name: "create", Label: "Create Lead", Method: "POST",
				Path: "/services/data/v58.0/sobjects/Lead", BodyParam: "body",
				Params: []schema.ParamSchema{body}},
			// Opportunities
			{Resource: "opportunity", Name: "list", Label: "List Opportunities", Method: "GET",
				Path: "/services/data/v58.0/query", ItemsPath: "records",
				Query:  map[string]string{"q": "=SELECT+Id,Name,Amount,StageName+FROM+Opportunity+LIMIT+"},
				Params: []schema.ParamSchema{limit}},
			// SOQL Query
			{Resource: "query", Name: "execute", Label: "Execute SOQL Query", Method: "GET",
				Path: "/services/data/v58.0/query", ItemsPath: "records",
				Query:  map[string]string{"q": "query"},
				Params: []schema.ParamSchema{query}},
			// Describe / metadata
			{Resource: "sobject", Name: "describe", Label: "Describe Object", Method: "GET",
				Path: "/services/data/v58.0/sobjects/{objectName}/describe",
				Params: []schema.ParamSchema{sp("objectName", "Object API Name", true)}},
		},
	}
}
