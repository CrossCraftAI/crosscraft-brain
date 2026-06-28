package accounting

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

// QuickBooks returns the QuickBooks Online REST API node (v3).
func QuickBooks(base string) rest.Node {
	body := jp("body", "Body (JSON)")
	limit := ip("limit", "Max results", 50)

	return rest.Node{
		Type: "accounting.quickbooks", Label: "QuickBooks Online", Group: "integration", Icon: "FileText",
		Description:  "Manage QuickBooks invoices, customers, expenses, and run reports.",
		BaseURL:      base,
		BaseURLParam: "baseUrl",
		CredType:     "quickbooksApi",
		Auth:         rest.Auth{Kind: "oauth2"},
		Ops: []rest.Op{
			// Invoices
			{Resource: "invoice", Name: "list", Label: "List Invoices", Method: "GET",
				Path: "/v3/company/{companyId}/query",
				Query:  map[string]string{"query": "=SELECT * FROM Invoice"},
				Params: []schema.ParamSchema{sp("companyId", "Company/Realm ID", true), limit}},
			{Resource: "invoice", Name: "get", Label: "Get Invoice", Method: "GET",
				Path: "/v3/company/{companyId}/invoice/{invoiceId}",
				Params: []schema.ParamSchema{sp("companyId", "Company ID", true), sp("invoiceId", "Invoice ID", true)}},
			{Resource: "invoice", Name: "create", Label: "Create Invoice", Method: "POST",
				Path: "/v3/company/{companyId}/invoice", BodyParam: "body",
				Params: []schema.ParamSchema{sp("companyId", "Company ID", true), body}},
			{Resource: "invoice", Name: "update", Label: "Update Invoice", Method: "POST",
				Path: "/v3/company/{companyId}/invoice", BodyParam: "body",
				Params: []schema.ParamSchema{sp("companyId", "Company ID", true), body}},
			// Customers
			{Resource: "customer", Name: "list", Label: "List Customers", Method: "GET",
				Path: "/v3/company/{companyId}/query",
				Query:  map[string]string{"query": "=SELECT * FROM Customer"},
				Params: []schema.ParamSchema{sp("companyId", "Company ID", true), limit}},
			{Resource: "customer", Name: "create", Label: "Create Customer", Method: "POST",
				Path: "/v3/company/{companyId}/customer", BodyParam: "body",
				Params: []schema.ParamSchema{sp("companyId", "Company ID", true), body}},
			{Resource: "customer", Name: "update", Label: "Update Customer", Method: "POST",
				Path: "/v3/company/{companyId}/customer", BodyParam: "body",
				Params: []schema.ParamSchema{sp("companyId", "Company ID", true), body}},
			// Expenses
			{Resource: "expense", Name: "list", Label: "List Expenses", Method: "GET",
				Path: "/v3/company/{companyId}/query",
				Query:  map[string]string{"query": "=SELECT * FROM Purchase"},
				Params: []schema.ParamSchema{sp("companyId", "Company ID", true), limit}},
			{Resource: "expense", Name: "create", Label: "Create Expense", Method: "POST",
				Path: "/v3/company/{companyId}/purchase", BodyParam: "body",
				Params: []schema.ParamSchema{sp("companyId", "Company ID", true), body}},
			// Reports
			{Resource: "report", Name: "profitLoss", Label: "Profit & Loss Report", Method: "GET",
				Path: "/v3/company/{companyId}/reports/ProfitAndLoss",
				Params: []schema.ParamSchema{sp("companyId", "Company ID", true)}},
			{Resource: "report", Name: "balanceSheet", Label: "Balance Sheet", Method: "GET",
				Path: "/v3/company/{companyId}/reports/BalanceSheet",
				Params: []schema.ParamSchema{sp("companyId", "Company ID", true)}},
		},
	}
}
