// Package commerce provides e-commerce integration nodes: Shopify and WooCommerce.
package commerce

import (
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// Nodes returns the full commerce node pack.
func Nodes() []schema.NodeDefinition {
	return []schema.NodeDefinition{
		Shopify("https://{store}.myshopify.com").Build(),
		WooCommerce("https://{store}").Build(),
	}
}
