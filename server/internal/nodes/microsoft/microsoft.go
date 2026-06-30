// Package microsoft provides Microsoft 365 integration nodes (Outlook, OneDrive,
// Teams, To Do, Excel, Calendar, SharePoint, OneNote, Graph) over the Microsoft
// Graph API, built on the declarative REST framework. They authenticate with a
// microsoftOAuth2Api credential; the engine's OAuth2 client provider injects +
// refreshes the token. One Graph base URL serves every service. Builders take the
// base so tests can point them at a mock server.
package microsoft

import (
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/rest"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

const (
	credType      = "microsoftOAuth2Api"
	graphBaseProd = "https://graph.microsoft.com/v1.0"
)

func node(base, typ, label, icon, desc string, ops []rest.Op) rest.Node {
	return rest.Node{
		Type: typ, Label: label, Group: "integration", Icon: icon, Description: desc,
		BaseURL: base, CredType: credType, Auth: rest.Auth{Kind: "oauth2"}, Ops: ops,
	}
}

func sp(name, label string, required bool) schema.ParamSchema {
	return schema.ParamSchema{Name: name, Label: label, Type: "string", Required: required}
}
func jp(name, label string) schema.ParamSchema {
	return schema.ParamSchema{Name: name, Label: label, Type: "json"}
}

// Nodes returns the Microsoft node pack wired to the production Graph endpoint.
func Nodes() []schema.NodeDefinition {
	return []schema.NodeDefinition{
		Outlook(graphBaseProd).Build(),
		OneDrive(graphBaseProd).Build(),
		Teams(graphBaseProd).Build(),
		ToDo(graphBaseProd).Build(),
		Excel(graphBaseProd).Build(),
		Calendar(graphBaseProd).Build(),
		SharePoint(graphBaseProd).Build(),
		OneNote(graphBaseProd).Build(),
		Graph(graphBaseProd).Build(),
	}
}
