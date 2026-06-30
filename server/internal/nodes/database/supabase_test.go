package database

import (
	"testing"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/credtype"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

func TestSupabaseNodeDefinition(t *testing.T) {
	def := SupabaseNode()
	if def.Type != "database.supabase" {
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

	opCount := countOperations(def)
	if opCount != 6 {
		t.Fatalf("expected 6 operations, got %d", opCount)
	}
}

func TestSupabaseNodeUsesRegisteredCredentialType(t *testing.T) {
	reg := credtype.Default()
	credType, ok := reg.Get("supabaseApi")
	if !ok {
		t.Fatal("expected supabaseApi credential type to be registered")
	}
	if credType.DisplayName == "" {
		t.Fatal("expected supabaseApi credential type to have a display name")
	}
	if len(credType.Fields) == 0 {
		t.Fatal("expected supabaseApi credential fields to be defined")
	}
}

func TestResolveSupabaseConfigRequiresCredential(t *testing.T) {
	ctx := &schema.ExecContext{
		Credential: func(paramName string) (map[string]any, error) {
			return nil, nil
		},
	}
	_, _, err := resolveSupabaseConfig(ctx)
	if err == nil {
		t.Fatal("expected error when credential is missing")
	}
}

func TestResolveSupabaseConfigFullConfig(t *testing.T) {
	ctx := &schema.ExecContext{
		Credential: func(paramName string) (map[string]any, error) {
			return map[string]any{
				"url":         "https://abcdef.supabase.co",
				"accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test",
			}, nil
		},
	}
	baseURL, apiKey, err := resolveSupabaseConfig(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if baseURL != "https://abcdef.supabase.co" {
		t.Fatalf("unexpected URL: %s", baseURL)
	}
	if apiKey != "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test" {
		t.Fatalf("unexpected API key: %s", apiKey)
	}
}

func TestSupabaseExecuteRequiresCredential(t *testing.T) {
	ctx := &schema.ExecContext{
		Params: map[string]any{
			"operation": "select",
			"table":     "users",
		},
		Credential: func(paramName string) (map[string]any, error) {
			return nil, nil
		},
	}
	_, err := executeSupabase(ctx)
	if err == nil {
		t.Fatal("expected error when credential is missing")
	}
}
