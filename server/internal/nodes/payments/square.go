package payments

import (
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/rest"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// Square — Square Payments REST API (v2). Auth: Bearer token via squareApi.
func Square(base string) rest.Node {
	body := jp("body", "Body (JSON)")
	paymentID := sp("paymentId", "Payment ID", true)
	orderID := sp("orderId", "Order ID", true)
	customerID := sp("customerId", "Customer ID", true)
	return rest.Node{
		Type: "payments.square", Label: "Square", Group: "integration", Icon: "CreditCard",
		Description:  "Process Square payments, manage orders, customers, and refunds.",
		BaseURL:      base,
		CredType:     "squareApi",
		Auth:         rest.Auth{Kind: "header", Header: "Authorization", Prefix: "Bearer ", ValueField: "accessToken"},
		Ops: []rest.Op{
			// Payments
			{Resource: "payment", Name: "create", Label: "Create Payment", Method: "POST",
				Path: "/v2/payments", BodyParam: "body",
				Params: []schema.ParamSchema{body}},
			{Resource: "payment", Name: "get", Label: "Get Payment", Method: "GET",
				Path: "/v2/payments/{paymentId}",
				Params: []schema.ParamSchema{paymentID}},
			{Resource: "payment", Name: "list", Label: "List Payments", Method: "GET",
				Path: "/v2/payments"},
			// Orders
			{Resource: "order", Name: "create", Label: "Create Order", Method: "POST",
				Path: "/v2/orders", BodyParam: "body",
				Params: []schema.ParamSchema{body}},
			{Resource: "order", Name: "get", Label: "Get Order", Method: "GET",
				Path: "/v2/orders/{orderId}",
				Params: []schema.ParamSchema{orderID}},
			{Resource: "order", Name: "update", Label: "Update Order", Method: "PUT",
				Path: "/v2/orders/{orderId}", BodyParam: "body",
				Params: []schema.ParamSchema{orderID, body}},
			// Customers
			{Resource: "customer", Name: "list", Label: "List Customers", Method: "GET",
				Path: "/v2/customers"},
			{Resource: "customer", Name: "create", Label: "Create Customer", Method: "POST",
				Path: "/v2/customers", BodyParam: "body",
				Params: []schema.ParamSchema{body}},
			{Resource: "customer", Name: "update", Label: "Update Customer", Method: "PUT",
				Path: "/v2/customers/{customerId}", BodyParam: "body",
				Params: []schema.ParamSchema{customerID, body}},
			// Refunds
			{Resource: "refund", Name: "create", Label: "Create Refund", Method: "POST",
				Path: "/v2/refunds", BodyParam: "body",
				Params: []schema.ParamSchema{body}},
			{Resource: "refund", Name: "get", Label: "Get Refund", Method: "GET",
				Path: "/v2/refunds/{refundId}",
				Params: []schema.ParamSchema{sp("refundId", "Refund ID", true)}},
			// Locations
			{Resource: "location", Name: "list", Label: "List Locations", Method: "GET",
				Path: "/v2/locations"},
		},
	}
}
