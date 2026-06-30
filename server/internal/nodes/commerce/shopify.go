package commerce

import (
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/rest"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

func sp(name, label string, required bool) schema.ParamSchema {
	return schema.ParamSchema{Name: name, Label: label, Type: "string", Required: required}
}
func jp(name, label string) schema.ParamSchema {
	return schema.ParamSchema{Name: name, Label: label, Type: "json"}
}
func ip(name, label string, def int) schema.ParamSchema {
	return schema.ParamSchema{Name: name, Label: label, Type: "number", Default: float64(def)}
}

// Shopify returns the Shopify Admin REST API node.
func Shopify(base string) rest.Node {
	body := jp("body", "Body (JSON)")
	limit := ip("limit", "Max results", 50)

	return rest.Node{
		Type: "commerce.shopify", Label: "Shopify", Group: "integration", Icon: "ShoppingBag",
		Description:  "Manage Shopify products, orders, customers, and draft orders.",
		BaseURL:      base,
		BaseURLParam: "baseUrl",
		CredType:     "shopifyApi",
		Auth:         rest.Auth{Kind: "header", Header: "X-Shopify-Access-Token", Prefix: "", ValueField: "accessToken"},
		Ops: []rest.Op{
			// Products
			{Resource: "product", Name: "list", Label: "List Products", Method: "GET",
				Path: "/admin/api/2024-04/products.json", ItemsPath: "products",
				Query:  map[string]string{"limit": "limit"},
				Params: []schema.ParamSchema{limit}},
			{Resource: "product", Name: "get", Label: "Get Product", Method: "GET",
				Path: "/admin/api/2024-04/products/{productId}.json", ItemsPath: "product",
				Params: []schema.ParamSchema{sp("productId", "Product ID", true)}},
			{Resource: "product", Name: "create", Label: "Create Product", Method: "POST",
				Path: "/admin/api/2024-04/products.json", BodyParam: "body",
				Params: []schema.ParamSchema{body}},
			{Resource: "product", Name: "update", Label: "Update Product", Method: "PUT",
				Path: "/admin/api/2024-04/products/{productId}.json", BodyParam: "body",
				Params: []schema.ParamSchema{sp("productId", "Product ID", true), body}},
			{Resource: "product", Name: "delete", Label: "Delete Product", Method: "DELETE",
				Path: "/admin/api/2024-04/products/{productId}.json",
				Params: []schema.ParamSchema{sp("productId", "Product ID", true)}},
			// Orders
			{Resource: "order", Name: "list", Label: "List Orders", Method: "GET",
				Path: "/admin/api/2024-04/orders.json", ItemsPath: "orders",
				Query:  map[string]string{"limit": "limit"},
				Params: []schema.ParamSchema{limit}},
			{Resource: "order", Name: "get", Label: "Get Order", Method: "GET",
				Path: "/admin/api/2024-04/orders/{orderId}.json", ItemsPath: "order",
				Params: []schema.ParamSchema{sp("orderId", "Order ID", true)}},
			{Resource: "order", Name: "update", Label: "Update Order", Method: "PUT",
				Path: "/admin/api/2024-04/orders/{orderId}.json", BodyParam: "body",
				Params: []schema.ParamSchema{sp("orderId", "Order ID", true), body}},
			// Customers
			{Resource: "customer", Name: "list", Label: "List Customers", Method: "GET",
				Path: "/admin/api/2024-04/customers.json", ItemsPath: "customers",
				Query:  map[string]string{"limit": "limit"},
				Params: []schema.ParamSchema{limit}},
			{Resource: "customer", Name: "create", Label: "Create Customer", Method: "POST",
				Path: "/admin/api/2024-04/customers.json", BodyParam: "body",
				Params: []schema.ParamSchema{body}},
			{Resource: "customer", Name: "update", Label: "Update Customer", Method: "PUT",
				Path: "/admin/api/2024-04/customers/{customerId}.json", BodyParam: "body",
				Params: []schema.ParamSchema{sp("customerId", "Customer ID", true), body}},
			// Draft Orders
			{Resource: "draftOrder", Name: "create", Label: "Create Draft Order", Method: "POST",
				Path: "/admin/api/2024-04/draft_orders.json", BodyParam: "body",
				Params: []schema.ParamSchema{body}},
			// Inventory
			{Resource: "inventory", Name: "list", Label: "List Inventory Levels", Method: "GET",
				Path: "/admin/api/2024-04/inventory_levels.json", ItemsPath: "inventory_levels"},
		},
	}
}
