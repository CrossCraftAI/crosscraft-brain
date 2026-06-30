package database

import (
	"database/sql"
	"testing"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/credtype"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

func TestMySQLNodeDefinition(t *testing.T) {
	def := MySQLNode()
	if def.Type != "database.mysql" {
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
	if opCount != 5 {
		t.Fatalf("expected 5 operations, got %d", opCount)
	}
}

func TestMySQLNodeUsesRegisteredCredentialType(t *testing.T) {
	reg := credtype.Default()
	credType, ok := reg.Get("mysqlApi")
	if !ok {
		t.Fatal("expected mysqlApi credential type to be registered")
	}
	if credType.DisplayName == "" {
		t.Fatal("expected mysqlApi credential type to have a display name")
	}
	if len(credType.Fields) == 0 {
		t.Fatal("expected mysqlApi credential fields to be defined")
	}
}

func TestResolveMySQLDSNRequiresCredential(t *testing.T) {
	ctx := &schema.ExecContext{
		Credential: func(paramName string) (map[string]any, error) {
			return nil, nil
		},
	}
	_, err := resolveMySQLDSN(ctx)
	if err == nil {
		t.Fatal("expected error when credential is missing")
	}
}

func TestResolveMySQLDSNFromFields(t *testing.T) {
	ctx := &schema.ExecContext{
		Credential: func(paramName string) (map[string]any, error) {
			return map[string]any{
				"host":     "db.example.com",
				"port":     "3307",
				"user":     "appuser",
				"password": "secret",
				"database": "mydb",
			}, nil
		},
	}
	dsn, err := resolveMySQLDSN(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "appuser:secret@tcp(db.example.com:3307)/mydb?parseTime=true"
	if dsn != expected {
		t.Fatalf("expected DSN %q, got %q", expected, dsn)
	}
}

func TestResolveMySQLDSNFromDSNString(t *testing.T) {
	ctx := &schema.ExecContext{
		Credential: func(paramName string) (map[string]any, error) {
			return map[string]any{
				"dsn": "user:pass@tcp(localhost:3306)/testdb?parseTime=true",
			}, nil
		},
	}
	dsn, err := resolveMySQLDSN(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dsn != "user:pass@tcp(localhost:3306)/testdb?parseTime=true" {
		t.Fatalf("unexpected DSN: %s", dsn)
	}
}

func TestMySQLExecuteRequiresDSN(t *testing.T) {
	ctx := &schema.ExecContext{
		Params: map[string]any{
			"operation": "query:many",
			"query":     "SELECT 1",
		},
		Credential: func(paramName string) (map[string]any, error) {
			return nil, nil
		},
	}
	_, err := executeMySQL(ctx)
	if err == nil {
		t.Fatal("expected error when credential is missing")
	}
}

func TestMySQLDBCachedByDSN(t *testing.T) {
	originalOpen := mysqlOpen
	mysqlDBMu.Lock()
	mysqlDBCache = make(map[string]*sql.DB)
	mysqlDBMu.Unlock()
	t.Cleanup(func() {
		mysqlOpen = originalOpen
		mysqlDBMu.Lock()
		mysqlDBCache = make(map[string]*sql.DB)
		mysqlDBMu.Unlock()
	})

	var createCalls int
	mysqlOpen = func(dsn string) (*sql.DB, error) {
		createCalls++
		return &sql.DB{}, nil
	}

	db1, err := getOrCreateMySQLDB("user:pass@tcp(localhost:3306)/testdb")
	if err != nil {
		t.Fatalf("expected first DB creation to succeed: %v", err)
	}
	db2, err := getOrCreateMySQLDB("user:pass@tcp(localhost:3306)/testdb")
	if err != nil {
		t.Fatalf("expected cached DB reuse to succeed: %v", err)
	}

	if createCalls != 1 {
		t.Fatalf("expected one DB creation, got %d", createCalls)
	}
	if db1 != db2 {
		t.Fatal("expected the same DB instance to be reused for the same DSN")
	}
}

func TestParseParams(t *testing.T) {
	// Test array input
	arr := parseParams([]any{"a", 1, true})
	if len(arr) != 3 {
		t.Fatalf("expected 3 params, got %d", len(arr))
	}

	// Test nil input
	nilParams := parseParams(nil)
	if nilParams != nil {
		t.Fatalf("expected nil for nil input")
	}

	// Test empty array
	empty := parseParams([]any{})
	if len(empty) != 0 {
		t.Fatalf("expected 0 params, got %d", len(empty))
	}
}
