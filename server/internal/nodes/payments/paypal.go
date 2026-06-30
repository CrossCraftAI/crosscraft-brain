package payments

import (
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/rest"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// PayPal — PayPal REST API (v2). Auth: OAuth2 Bearer via paypalApi credential.
func PayPal(base string) rest.Node {
	body := jp("body", "Body (JSON)")
	orderID := sp("orderId", "Order ID", true)
	paymentID := sp("paymentId", "Payment ID", true)
	refundID := sp("refundId", "Refund ID", true)

	return rest.Node{
		Type: "payments.paypal", Label: "PayPal", Group: "integration", Icon: "DollarSign",
		Description:  "Create and manage PayPal orders, payments, refunds, and webhooks.",
		BaseURL:      base,
		CredType:     "paypalApi",
		Auth:         rest.Auth{Kind: "oauth2"},
		Ops: []rest.Op{
			// Orders
			{Resource: "order", Name: "create", Label: "Create Order", Method: "POST",
				Path: "/v2/checkout/orders", BodyParam: "body",
				Params: []schema.ParamSchema{body}},
			{Resource: "order", Name: "get", Label: "Get Order", Method: "GET",
				Path: "/v2/checkout/orders/{orderId}",
				Params: []schema.ParamSchema{orderID}},
			{Resource: "order", Name: "capture", Label: "Capture Order", Method: "POST",
				Path: "/v2/checkout/orders/{orderId}/capture", BodyParam: "body",
				Params: []schema.ParamSchema{orderID, body}},
			// Payments
			{Resource: "payment", Name: "get", Label: "Get Payment (Capture)", Method: "GET",
				Path: "/v2/payments/captures/{paymentId}",
				Params: []schema.ParamSchema{paymentID}},
			{Resource: "payment", Name: "authorization", Label: "Get Authorization", Method: "GET",
				Path: "/v2/payments/authorizations/{paymentId}",
				Params: []schema.ParamSchema{sp("authorizationId", "Authorization ID", true)}},
			{Resource: "payment", Name: "captureAuth", Label: "Capture Authorization", Method: "POST",
				Path: "/v2/payments/authorizations/{paymentId}/capture", BodyParam: "body",
				Params: []schema.ParamSchema{paymentID, body}},
			// Refunds
			{Resource: "refund", Name: "create", Label: "Create Refund", Method: "POST",
				Path: "/v2/payments/captures/{paymentId}/refund", BodyParam: "body",
				Params: []schema.ParamSchema{paymentID, body}},
			{Resource: "refund", Name: "get", Label: "Get Refund", Method: "GET",
				Path: "/v2/payments/refunds/{refundId}",
				Params: []schema.ParamSchema{refundID}},
			// Webhooks
			{Resource: "webhook", Name: "list", Label: "List Webhooks", Method: "GET",
				Path: "/v1/notifications/webhooks", ItemsPath: "webhooks"},
			{Resource: "webhook", Name: "create", Label: "Create Webhook", Method: "POST",
				Path: "/v1/notifications/webhooks", BodyParam: "body",
				Params: []schema.ParamSchema{body}},
			// Invoicing
			{Resource: "invoice", Name: "create", Label: "Create Invoice", Method: "POST",
				Path: "/v2/invoicing/invoices", BodyParam: "body",
				Params: []schema.ParamSchema{body}},
			{Resource: "invoice", Name: "list", Label: "List Invoices", Method: "GET",
				Path: "/v2/invoicing/invoices", ItemsPath: "items"},
		},
	}
}
