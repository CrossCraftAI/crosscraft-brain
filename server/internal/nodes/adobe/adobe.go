// Package adobe provides Adobe integration nodes: Acrobat Sign, PDF Services,
// Firefly, Photoshop, Lightroom, AEM Assets, Analytics, Stock, and Commerce.
// Adobe ships no official Go SDKs, so these are declarative REST nodes.
// Acrobat Sign uses a Bearer token (Integration Key). Most Creative Cloud APIs
// (PDF Services, Firefly, Photoshop, Lightroom, AEM, Analytics, Stock) use
// Adobe IMS server-to-server OAuth2 via adobeOAuth2Api. Commerce (Magento)
// uses an access token via adobeCommerceApi.
package adobe

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

func signNode(base, cred string, ops []rest.Op) rest.Node {
	return rest.Node{
		BaseURL: base, CredType: cred,
		Auth:   rest.Auth{Kind: "header", Header: "Authorization", Prefix: "Bearer ", ValueField: "accessToken"},
		Ops:    ops,
	}
}

func oauth2Node(base, typ, label, icon, desc string, ops []rest.Op) rest.Node {
	return rest.Node{
		Type: typ, Label: label, Group: "integration", Icon: icon, Description: desc,
		BaseURL: base, CredType: "adobeOAuth2Api",
		Auth: rest.Auth{Kind: "oauth2"}, Ops: ops,
	}
}

// Nodes returns the Adobe node pack.
func Nodes() []schema.NodeDefinition {
	return []schema.NodeDefinition{
		AcrobatSign("https://api.na1.adobesign.com/api/rest/v6").Build(),
		PDFServices("https://pdf-services.adobe.io").Build(),
		Firefly("https://firefly-api.adobe.io").Build(),
		Photoshop("https://image.adobe.io").Build(),
		Lightroom("https://lr.adobe.io").Build(),
		AEMAssets("").Build(),
		Analytics("https://analytics.adobe.io").Build(),
		Stock("https://stock.adobe.io").Build(),
		Commerce("").Build(),
	}
}
