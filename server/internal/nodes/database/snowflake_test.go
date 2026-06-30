package database

import (
	"database/sql"
	"testing"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/credtype"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

func TestSnowflakeNodeDefinition(t *testing.T) {
	def := SnowflakeNode()
	if def.Type != "database.snowflake" {
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
	if opCount != 4 {
		t.Fatalf("expected 4 operations, got %d", opCount)
	}
}

func TestSnowflakeNodeUsesRegisteredCredentialType(t *testing.T) {
	reg := credtype.Default()
	credType, ok := reg.Get("snowflakeApi")
	if !ok {
		t.Fatal("expected snowflakeApi credential type to be registered")
	}
	if credType.DisplayName == "" {
		t.Fatal("expected snowflakeApi credential type to have a display name")
	}
	if len(credType.Fields) == 0 {
		t.Fatal("expected snowflakeApi credential fields to be defined")
	}
}

func TestResolveSnowflakeDSNRequiresCredential(t *testing.T) {
	ctx := &schema.ExecContext{
		Credential: func(paramName string) (map[string]any, error) {
			return nil, nil
		},
	}
	_, err := resolveSnowflakeDSN(ctx)
	if err == nil {
		t.Fatal("expected error when credential is missing")
	}
}

func TestResolveSnowflakeDSNFullConfig(t *testing.T) {
	ctx := &schema.ExecContext{
		Credential: func(paramName string) (map[string]any, error) {
			return map[string]any{
				"account":   "myorg-abc123",
				"user":      "analyst",
				"password":  "secret",
				"warehouse": "COMPUTE_WH",
				"database":  "ANALYTICS",
				"schema":    "PUBLIC",
			}, nil
		},
	}
	dsn, err := resolveSnowflakeDSN(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "analyst:secret@myorg-abc123/ANALYTICS/PUBLIC?warehouse=COMPUTE_WH"
	if dsn != expected {
		t.Fatalf("expected DSN %q, got %q", expected, dsn)
	}
}

func TestSnowflakeDBCached(t *testing.T) {
	snowflakeDBMu.Lock()
	snowflakeDBCache = make(map[string]*sql.DB)
	snowflakeDBMu.Unlock()
	t.Cleanup(func() {
		snowflakeDBMu.Lock()
		snowflakeDBCache = make(map[string]*sql.DB)
		snowflakeDBMu.Unlock()
	})
}

func TestSnowflakeExecuteRequiresCredential(t *testing.T) {
	ctx := &schema.ExecContext{
		Params: map[string]any{
			"operation": "query:many",
			"query":     "SELECT 1",
		},
		Credential: func(paramName string) (map[string]any, error) {
			return nil, nil
		},
	}
	_, err := executeSnowflake(ctx)
	if err == nil {
		t.Fatal("expected error when credential is missing")
	}
}
