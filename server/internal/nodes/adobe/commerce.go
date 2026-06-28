package adobe

import (
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/rest"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// Commerce — Adobe Commerce (Magento) REST API. Base URL is user-overridable
// (the store's domain). Auth uses an access token via adobeCommerceApi.
func Commerce(base string) rest.Node {
	body := jp("body", "Body (JSON)")
	customerID := sp("customerId", "Customer ID", true)
	productSKU := sp("sku", "Product SKU", true)
	orderID := sp("orderId", "Order ID", true)
	invoiceID := sp("invoiceId", "Invoice ID", true)
	searchQ := schema.ParamSchema{Name: "searchCriteria", Label: "Search criteria (JSON)", Type: "json"}
	limit := ip("pageSize", "Page size", 50)

	return rest.Node{
		Type: "adobe.commerce", Label: "Adobe Commerce (Magento)", Group: "integration", Icon: "ShoppingCart",
		Description:  "Manage Magento customers, products, orders, and invoices.",
		BaseURL:      base,
		BaseURLParam: "baseUrl",
		CredType:     "adobeCommerceApi",
		Auth:         rest.Auth{Kind: "header", Header: "Authorization", Prefix: "Bearer ", ValueField: "accessToken"},
		Ops: []rest.Op{
			// Customers
			{Resource: "customer", Name: "list", Label: "List Customers", Method: "GET",
				Path: "/rest/V1/customers/search", ItemsPath: "items",
				Query: map[string]string{"searchCriteria[pageSize]": "pageSize"},
				Params: []schema.ParamSchema{limit, searchQ}},
			{Resource: "customer", Name: "get", Label: "Get Customer", Method: "GET",
				Path: "/rest/V1/customers/{customerId}",
				Params: []schema.ParamSchema{customerID}},
			{Resource: "customer", Name: "create", Label: "Create Customer", Method: "POST",
				Path: "/rest/V1/customers", BodyParam: "body",
				Params: []schema.ParamSchema{body}},
			{Resource: "customer", Name: "update", Label: "Update Customer", Method: "PUT",
				Path: "/rest/V1/customers/{customerId}", BodyParam: "body",
				Params: []schema.ParamSchema{customerID, body}},
			// Products
			{Resource: "product", Name: "list", Label: "List Products", Method: "GET",
				Path: "/rest/V1/products", ItemsPath: "items",
				Query: map[string]string{"searchCriteria[pageSize]": "pageSize"},
				Params: []schema.ParamSchema{limit, searchQ}},
			{Resource: "product", Name: "get", Label: "Get Product", Method: "GET",
				Path: "/rest/V1/products/{sku}",
				Params: []schema.ParamSchema{productSKU}},
			{Resource: "product", Name: "create", Label: "Create Product", Method: "POST",
				Path: "/rest/V1/products", BodyParam: "body",
				Params: []schema.ParamSchema{body}},
			{Resource: "product", Name: "update", Label: "Update Product", Method: "PUT",
				Path: "/rest/V1/products/{sku}", BodyParam: "body",
				Params: []schema.ParamSchema{productSKU, body}},
			// Orders
			{Resource: "order", Name: "list", Label: "List Orders", Method: "GET",
				Path: "/rest/V1/orders", ItemsPath: "items",
				Query: map[string]string{"searchCriteria[pageSize]": "pageSize"},
				Params: []schema.ParamSchema{limit, searchQ}},
			{Resource: "order", Name: "get", Label: "Get Order", Method: "GET",
				Path: "/rest/V1/orders/{orderId}",
				Params: []schema.ParamSchema{orderID}},
			{Resource: "order", Name: "create", Label: "Create Order", Method: "POST",
				Path: "/rest/V1/orders", BodyParam: "body",
				Params: []schema.ParamSchema{body}},
			{Resource: "order", Name: "cancel", Label: "Cancel Order", Method: "POST",
				Path: "/rest/V1/orders/{orderId}/cancel",
				Params: []schema.ParamSchema{orderID}},
			// Invoices
			{Resource: "invoice", Name: "list", Label: "List Invoices", Method: "GET",
				Path: "/rest/V1/invoices", ItemsPath: "items",
				Query: map[string]string{"searchCriteria[pageSize]": "pageSize"},
				Params: []schema.ParamSchema{limit, searchQ}},
			{Resource: "invoice", Name: "get", Label: "Get Invoice", Method: "GET",
				Path: "/rest/V1/invoices/{invoiceId}",
				Params: []schema.ParamSchema{invoiceID}},
			{Resource: "invoice", Name: "create", Label: "Create Invoice", Method: "POST",
				Path: "/rest/V1/invoices", BodyParam: "body",
				Params: []schema.ParamSchema{body}},
			// Store info
			{Resource: "store", Name: "list", Label: "List Store Views", Method: "GET",
				Path: "/rest/V1/store/storeViews"},
			{Resource: "store", Name: "config", Label: "Get Store Config", Method: "GET",
				Path: "/rest/V1/store/storeConfigs"},
		},
	}
}
