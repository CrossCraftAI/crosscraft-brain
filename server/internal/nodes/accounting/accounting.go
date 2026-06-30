// Package accounting provides accounting integration nodes: QuickBooks Online and Xero.
package accounting

import (
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// Nodes returns the full accounting node pack.
func Nodes() []schema.NodeDefinition {
	return []schema.NodeDefinition{
		QuickBooks("https://quickbooks.api.intuit.com").Build(),
		Xero("https://api.xero.com").Build(),
	}
}
