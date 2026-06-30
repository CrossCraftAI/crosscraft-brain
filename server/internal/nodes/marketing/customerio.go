package marketing

import (
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/rest"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// CustomerIO — Customer.io messaging and automation platform.
// App API (v2): customer & campaign management; Track API (v1): event tracking.
// Auth: Bearer token (App API Key for v2, or Site ID + API Key for v1).
// Use baseUrl param to switch between App API and Track API endpoints.
func CustomerIO() rest.Node {
	customerID := sp("customerId", "Customer ID", true)
	campaignID := sp("campaignId", "Campaign ID", true)
	body := jp("body", "Body (JSON)")
	limit := ip("limit", "Max results", 100)

	return rest.Node{
		Type: "marketing.customerio", Label: "Customer.io", Group: "integration", Icon: "Users",
		Description:  "Manage Customer.io customers, campaigns, messages, and track events.",
		BaseURL:      "https://api.customer.io/v2",
		BaseURLParam: "baseUrl",
		CredType:     "customerioApi",
		Auth:         rest.Auth{Kind: "header", Header: "Authorization", Prefix: "Bearer ", ValueField: "accessToken"},
		Ops: []rest.Op{
			// Customers (App API v2)
			{Resource: "customer", Name: "list", Label: "List Customers", Method: "GET",
				Path: "/customers", ItemsPath: "customers",
				Query:  map[string]string{"limit": "limit"},
				Params: []schema.ParamSchema{limit}},
			{Resource: "customer", Name: "create", Label: "Create / Update Customer", Method: "PUT",
				Path: "/customers/{customerId}", BodyParam: "body",
				Params: []schema.ParamSchema{customerID, body}},
			{Resource: "customer", Name: "delete", Label: "Delete Customer", Method: "DELETE",
				Path: "/customers/{customerId}",
				Params: []schema.ParamSchema{customerID}},
			// Event tracking (uses Track API v1 — override baseUrl to https://track.customer.io/api/v1)
			{Resource: "event", Name: "track", Label: "Track Event", Method: "POST",
				Path: "/customers/{customerId}/events", BodyParam: "body",
				Params: []schema.ParamSchema{customerID, body}},
			// Campaigns (App API v2)
			{Resource: "campaign", Name: "list", Label: "List Campaigns", Method: "GET",
				Path: "/campaigns", ItemsPath: "campaigns",
				Params: []schema.ParamSchema{limit}},
			{Resource: "campaign", Name: "trigger", Label: "Trigger Campaign", Method: "POST",
				Path: "/campaigns/{campaignId}/triggers", BodyParam: "body",
				Params: []schema.ParamSchema{campaignID, body}},
			// Messages (App API v2)
			{Resource: "message", Name: "list", Label: "List Messages", Method: "GET",
				Path: "/messages", ItemsPath: "messages",
				Query:  map[string]string{"limit": "limit"},
				Params: []schema.ParamSchema{limit}},
		},
	}
}
