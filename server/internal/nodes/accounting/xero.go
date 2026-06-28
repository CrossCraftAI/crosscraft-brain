package accounting

import (
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/rest"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// Xero returns the Xero Accounting REST API node (v2.0).
func Xero(base string) rest.Node {
	body := jp("body", "Body (JSON)")

	return rest.Node{
		Type: "accounting.xero", Label: "Xero", Group: "integration", Icon: "Calculator",
		Description:  "Manage Xero invoices, contacts, bank transactions, and run reports.",
		BaseURL:      base,
		CredType:     "xeroApi",
		Auth:         rest.Auth{Kind: "oauth2"},
		Ops: []rest.Op{
			// Invoices
			{Resource: "invoice", Name: "list", Label: "List Invoices", Method: "GET",
				Path: "/api.xro/2.0/Invoices", ItemsPath: "Invoices"},
			{Resource: "invoice", Name: "get", Label: "Get Invoice", Method: "GET",
				Path: "/api.xro/2.0/Invoices/{invoiceId}",
				Params: []schema.ParamSchema{sp("invoiceId", "Invoice ID", true)}},
			{Resource: "invoice", Name: "create", Label: "Create Invoice", Method: "POST",
				Path: "/api.xro/2.0/Invoices", BodyParam: "body",
				Params: []schema.ParamSchema{body}},
			{Resource: "invoice", Name: "update", Label: "Update Invoice", Method: "POST",
				Path: "/api.xro/2.0/Invoices/{invoiceId}", BodyParam: "body",
				Params: []schema.ParamSchema{sp("invoiceId", "Invoice ID", true), body}},
			// Contacts
			{Resource: "contact", Name: "list", Label: "List Contacts", Method: "GET",
				Path: "/api.xro/2.0/Contacts", ItemsPath: "Contacts"},
			{Resource: "contact", Name: "create", Label: "Create Contact", Method: "POST",
				Path: "/api.xro/2.0/Contacts", BodyParam: "body",
				Params: []schema.ParamSchema{body}},
			{Resource: "contact", Name: "update", Label: "Update Contact", Method: "POST",
				Path: "/api.xro/2.0/Contacts/{contactId}", BodyParam: "body",
				Params: []schema.ParamSchema{sp("contactId", "Contact ID", true), body}},
			// Bank Transactions
			{Resource: "bankTransaction", Name: "list", Label: "List Bank Transactions", Method: "GET",
				Path: "/api.xro/2.0/BankTransactions", ItemsPath: "BankTransactions"},
			{Resource: "bankTransaction", Name: "create", Label: "Create Bank Transaction", Method: "POST",
				Path: "/api.xro/2.0/BankTransactions", BodyParam: "body",
				Params: []schema.ParamSchema{body}},
			// Reports
			{Resource: "report", Name: "profitLoss", Label: "Profit & Loss Report", Method: "GET",
				Path: "/api.xro/2.0/Reports/ProfitAndLoss"},
			{Resource: "report", Name: "balanceSheet", Label: "Balance Sheet Report", Method: "GET",
				Path: "/api.xro/2.0/Reports/BalanceSheet"},
			// Accounts
			{Resource: "account", Name: "list", Label: "List Chart of Accounts", Method: "GET",
				Path: "/api.xro/2.0/Accounts", ItemsPath: "Accounts"},
		},
	}
}
