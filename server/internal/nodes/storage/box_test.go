package storage

import (
	"testing"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/credtype"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

func TestBoxNodeDefinition(t *testing.T) {
	def := BoxNode()
	if def.Type != "storage.box" {
		t.Fatalf("unexpected type: %s", def.Type)
	}
	if len(def.Params) == 0 {
		t.Fatal("expected params to be defined")
	}
	if def.Execute == nil {
		t.Fatal("expected execute function to be set")
	}
	if len(def.Outputs) != 2 {
		t.Fatalf("expected 2 outputs, got %d", len(def.Outputs))
	}

	opCount := countStorageOperations(def)
	if opCount != 8 {
		t.Fatalf("expected 8 operations, got %d", opCount)
	}
}

func TestBoxNodeUsesRegisteredCredentialType(t *testing.T) {
	reg := credtype.Default()
	credType, ok := reg.Get("boxApi")
	if !ok {
		t.Fatal("expected boxApi credential type to be registered")
	}
	if credType.DisplayName == "" {
		t.Fatal("expected boxApi credential type to have a display name")
	}
	if len(credType.Fields) == 0 {
		t.Fatal("expected boxApi credential fields to be defined")
	}
}

func TestBoxExecuteRequiresCredential(t *testing.T) {
	ctx := &schema.ExecContext{
		Params: map[string]any{
			"operation": "list",
			"folderId":  "0",
		},
		Credential: func(paramName string) (map[string]any, error) {
			return nil, nil
		},
	}
	_, err := executeBox(ctx)
	if err == nil {
		t.Fatal("expected error when credential is missing")
	}
}

func TestBoxExecuteRequiresToken(t *testing.T) {
	ctx := &schema.ExecContext{
		Params: map[string]any{
			"operation": "list",
			"folderId":  "0",
		},
		Credential: func(paramName string) (map[string]any, error) {
			return map[string]any{"accessToken": ""}, nil
		},
	}
	_, err := executeBox(ctx)
	if err == nil {
		t.Fatal("expected error when access token is empty")
	}
}
