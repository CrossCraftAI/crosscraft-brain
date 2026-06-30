// Package marketing provides marketing-automation integration nodes: Customer.io,
// ActiveCampaign, and Brevo (Sendinblue). All use the declarative REST framework.
package marketing

import (
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

func sp(name, label string, required bool) schema.ParamSchema {
	return schema.ParamSchema{Name: name, Label: label, Type: "string", Required: required}
}
func ip(name, label string, def int) schema.ParamSchema {
	return schema.ParamSchema{Name: name, Label: label, Type: "number", Default: def}
}
func jp(name, label string) schema.ParamSchema {
	return schema.ParamSchema{Name: name, Label: label, Type: "json"}
}

// Nodes returns the full marketing-automation node pack.
func Nodes() []schema.NodeDefinition {
	return []schema.NodeDefinition{
		CustomerIO().Build(),
		ActiveCampaign().Build(),
		Brevo().Build(),
	}
}

