package commerce

import (
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/rest"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// WooCommerce returns the WooCommerce REST API node (v3).
// Auth: OAuth1/Consumer Key via query parameters (simplified as Basic auth here).
func WooCommerce(base string) rest.Node {
	body := jp("body", "Body (JSON)")
	limit := ip("per_page", "Per page", 50)

	return rest.Node{
		Type: "commerce.woocommerce", Label: "WooCommerce", Group: "integration", Icon: "ShoppingCart",
		Description:  "Manage WooCommerce products, orders, customers, and coupons.",
		BaseURL:      base,
		BaseURLParam: "baseUrl",
		CredType:     "woocommerceApi",
		Auth:         rest.Auth{Kind: "header", Header: "Authorization", Prefix: "Basic ", ValueField: "accessToken"},
		Ops: []rest.Op{
			// Products
			{Resource: "product", Name: "list", Label: "List Products", Method: "GET",
				Path: "/wp-json/wc/v3/products",
				Query:  map[string]string{"per_page": "per_page"},
				Params: []schema.ParamSchema{limit}},
			{Resource: "product", Name: "create", Label: "Create Product", Method: "POST",
				Path: "/wp-json/wc/v3/products", BodyParam: "body",
				Params: []schema.ParamSchema{body}},
			{Resource: "product", Name: "update", Label: "Update Product", Method: "PUT",
				Path: "/wp-json/wc/v3/products/{productId}", BodyParam: "body",
				Params: []schema.ParamSchema{sp("productId", "Product ID", true), body}},
			{Resource: "product", Name: "delete", Label: "Delete Product", Method: "DELETE",
				Path: "/wp-json/wc/v3/products/{productId}",
				Params: []schema.ParamSchema{sp("productId", "Product ID", true)}},
			// Orders
			{Resource: "order", Name: "list", Label: "List Orders", Method: "GET",
				Path: "/wp-json/wc/v3/orders",
				Query:  map[string]string{"per_page": "per_page"},
				Params: []schema.ParamSchema{limit}},
			{Resource: "order", Name: "get", Label: "Get Order", Method: "GET",
				Path: "/wp-json/wc/v3/orders/{orderId}",
				Params: []schema.ParamSchema{sp("orderId", "Order ID", true)}},
			{Resource: "order", Name: "update", Label: "Update Order", Method: "PUT",
				Path: "/wp-json/wc/v3/orders/{orderId}", BodyParam: "body",
				Params: []schema.ParamSchema{sp("orderId", "Order ID", true), body}},
			// Customers
			{Resource: "customer", Name: "list", Label: "List Customers", Method: "GET",
				Path: "/wp-json/wc/v3/customers",
				Query:  map[string]string{"per_page": "per_page"},
				Params: []schema.ParamSchema{limit}},
			{Resource: "customer", Name: "create", Label: "Create Customer", Method: "POST",
				Path: "/wp-json/wc/v3/customers", BodyParam: "body",
				Params: []schema.ParamSchema{body}},
			{Resource: "customer", Name: "update", Label: "Update Customer", Method: "PUT",
				Path: "/wp-json/wc/v3/customers/{customerId}", BodyParam: "body",
				Params: []schema.ParamSchema{sp("customerId", "Customer ID", true), body}},
			// Coupons
			{Resource: "coupon", Name: "list", Label: "List Coupons", Method: "GET",
				Path: "/wp-json/wc/v3/coupons",
				Query:  map[string]string{"per_page": "per_page"},
				Params: []schema.ParamSchema{limit}},
			{Resource: "coupon", Name: "create", Label: "Create Coupon", Method: "POST",
				Path: "/wp-json/wc/v3/coupons", BodyParam: "body",
				Params: []schema.ParamSchema{body}},
			// Reports
			{Resource: "report", Name: "sales", Label: "Sales Report", Method: "GET",
				Path: "/wp-json/wc/v3/reports/sales"},
			{Resource: "report", Name: "orders", Label: "Orders Report", Method: "GET",
				Path: "/wp-json/wc/v3/reports/orders/totals"},
		},
	}
}
