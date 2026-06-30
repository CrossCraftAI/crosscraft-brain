// Package storage provides cloud storage integration nodes: Dropbox and Box.
package storage

import "github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"

// Nodes returns all storage integration nodes.
func Nodes() []schema.NodeDefinition {
	return []schema.NodeDefinition{
		DropboxNode(),
		BoxNode(),
	}
}
