package database

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
	_ "github.com/snowflakedb/gosnowflake"
)

var (
	snowflakeDBMu    sync.Mutex
	snowflakeDBCache = make(map[string]*sql.DB)
	snowflakeOpen    = func(dsn string) (*sql.DB, error) {
		db, err := sql.Open("snowflake", dsn)
		if err != nil {
			return nil, err
		}
		if err := db.Ping(); err != nil {
			return nil, fmt.Errorf("snowflake: ping failed: %w", err)
		}
		return db, nil
	}
)

// SnowflakeNode returns the definition for the Snowflake node.
func SnowflakeNode() schema.NodeDefinition {
	return schema.NodeDefinition{
		Type:        "database.snowflake",
		Label:       "Snowflake",
		Description: "Query and execute commands on a Snowflake data warehouse.",
		Group:       "storage",
		Icon:        "Database",
		Inputs:      []schema.Port{{ID: "main"}},
		Outputs:     []schema.Port{{ID: "main", Label: "Results"}, {ID: "error", Label: "Error"}},
		Credentials: []string{"snowflakeApi"},
		Params: []schema.ParamSchema{
			{Name: "credential", Label: "Credential", Type: "credential", Required: true, CredentialType: "snowflakeApi"},
			{Name: "operation", Label: "Operation", Type: "select", Required: true, Default: "query:many", Options: []schema.ParamOption{
				{Label: "Query (multiple rows)", Value: "query:many"},
				{Label: "Query (single row)", Value: "query:one"},
				{Label: "Execute (Insert, Update, Delete)", Value: "exec"},
				{Label: "Trigger: New Row", Value: "trigger:newRow"},
			}},
			{Name: "query", Label: "SQL Query", Type: "code:sql", Required: true,
				Description: "SQL statement. Use ? for parameterized queries.",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"query:many", "query:one", "exec"}}},
			{Name: "params", Label: "Parameters (JSON array)", Type: "json",
				Description: "Array of parameter values for ? placeholders."},
			{Name: "table", Label: "Table Name", Type: "string",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"trigger:newRow"}}},
			{Name: "idColumn", Label: "ID Column", Type: "string", Default: "id",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"trigger:newRow"}}},
		},
		Execute: executeSnowflake,
	}
}

// resolveSnowflakeDSN builds a Snowflake DSN from the credential.
// Format: user:password@account/database/schema?warehouse=warehouse
func resolveSnowflakeDSN(ctx *schema.ExecContext) (string, error) {
	if ctx.Credential != nil {
		cred, err := ctx.Credential("credential")
		if err != nil {
			return "", fmt.Errorf("snowflake: failed to get credentials: %w", err)
		}
		if len(cred) > 0 {
			account, _ := cred["account"].(string)
			user, _ := cred["user"].(string)
			password, _ := cred["password"].(string)
			warehouse, _ := cred["warehouse"].(string)
			database, _ := cred["database"].(string)
			schema, _ := cred["schema"].(string)

			if account == "" || user == "" || password == "" {
				return "", fmt.Errorf("snowflake: account, user, and password are required")
			}

			dsn := fmt.Sprintf("%s:%s@%s", user, password, account)
			if database != "" {
				dsn += "/" + database
				if schema != "" {
					dsn += "/" + schema
				}
			}
			if warehouse != "" {
				dsn += "?warehouse=" + warehouse
			}
			return dsn, nil
		}
	}
	return "", fmt.Errorf("snowflake: no credential configured")
}

// getOrCreateSnowflakeDB returns a cached connection pool for the given DSN.
func getOrCreateSnowflakeDB(dsn string) (*sql.DB, error) {
	snowflakeDBMu.Lock()
	if db, ok := snowflakeDBCache[dsn]; ok {
		snowflakeDBMu.Unlock()
		return db, nil
	}
	snowflakeDBMu.Unlock()

	db, err := snowflakeOpen(dsn)
	if err != nil {
		return nil, fmt.Errorf("snowflake: failed to open: %w", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	snowflakeDBMu.Lock()
	defer snowflakeDBMu.Unlock()
	if existing, ok := snowflakeDBCache[dsn]; ok {
		db.Close()
		return existing, nil
	}
	snowflakeDBCache[dsn] = db
	return db, nil
}

// executeSnowflake is the execution function for the Snowflake node.
func executeSnowflake(ctx *schema.ExecContext) (schema.NodeResult, error) {
	dsn, err := resolveSnowflakeDSN(ctx)
	if err != nil {
		return schema.NodeResult{}, err
	}

	db, err := getOrCreateSnowflakeDB(dsn)
	if err != nil {
		return schema.NodeResult{}, err
	}

	operation, _ := ctx.Params["operation"].(string)
	execCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second) // Snowflake queries can be slower
	defer cancel()

	switch operation {
	case "query:many":
		return snowflakeQueryMany(execCtx, db, ctx)
	case "query:one":
		return snowflakeQueryOne(execCtx, db, ctx)
	case "exec":
		return snowflakeExec(execCtx, db, ctx)
	case "trigger:newRow":
		return snowflakeTriggerNewRow(execCtx, db, ctx)
	default:
		return schema.NodeResult{}, fmt.Errorf("snowflake: unknown operation %q", operation)
	}
}

func snowflakeQueryMany(ctx context.Context, db *sql.DB, execCtx *schema.ExecContext) (schema.NodeResult, error) {
	query, _ := execCtx.Params["query"].(string)
	if query == "" {
		return schema.NodeResult{}, fmt.Errorf("snowflake: query is required")
	}
	params := parseParams(execCtx.RawParam("params"))

	rows, err := db.QueryContext(ctx, query, params...)
	if err != nil {
		return schema.NodeResult{}, fmt.Errorf("snowflake query failed: %w", err)
	}
	defer rows.Close()

	var out []schema.Item
	for rows.Next() {
		rowData, err := scanRow(rows)
		if err != nil {
			continue
		}
		out = append(out, schema.Item{JSON: rowData})
	}
	if err := rows.Err(); err != nil {
		return schema.NodeResult{}, fmt.Errorf("snowflake rows error: %w", err)
	}
	if out == nil {
		out = []schema.Item{}
	}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": out}}, nil
}

func snowflakeQueryOne(ctx context.Context, db *sql.DB, execCtx *schema.ExecContext) (schema.NodeResult, error) {
	query, _ := execCtx.Params["query"].(string)
	if query == "" {
		return schema.NodeResult{}, fmt.Errorf("snowflake: query is required")
	}
	params := parseParams(execCtx.RawParam("params"))

	rows, err := db.QueryContext(ctx, query, params...)
	if err != nil {
		return schema.NodeResult{}, fmt.Errorf("snowflake query failed: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		rowData, err := scanRow(rows)
		if err != nil {
			return schema.NodeResult{}, fmt.Errorf("snowflake row scan failed: %w", err)
		}
		return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{JSON: rowData}}}}, nil
	}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {}}}, nil
}

func snowflakeExec(ctx context.Context, db *sql.DB, execCtx *schema.ExecContext) (schema.NodeResult, error) {
	query, _ := execCtx.Params["query"].(string)
	if query == "" {
		return schema.NodeResult{}, fmt.Errorf("snowflake: query is required")
	}
	params := parseParams(execCtx.RawParam("params"))

	result, err := db.ExecContext(ctx, query, params...)
	if err != nil {
		return schema.NodeResult{}, fmt.Errorf("snowflake exec failed: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	lastInsertID, _ := result.LastInsertId()
	out := []schema.Item{{JSON: map[string]any{
		"status":       "success",
		"rowsAffected": rowsAffected,
		"lastInsertId": lastInsertID,
	}}}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": out}}, nil
}

func snowflakeTriggerNewRow(ctx context.Context, db *sql.DB, execCtx *schema.ExecContext) (schema.NodeResult, error) {
	table, _ := execCtx.Params["table"].(string)
	if table == "" {
		return schema.NodeResult{}, fmt.Errorf("snowflake trigger: table name is required")
	}
	idColumn, _ := execCtx.Params["idColumn"].(string)
	if idColumn == "" {
		idColumn = "id"
	}

	lastID, _ := execCtx.State["lastId"].(string)
	query := fmt.Sprintf("SELECT * FROM %s", table)
	if lastID != "" {
		query += fmt.Sprintf(" WHERE %s > ?", idColumn)
	}
	query += fmt.Sprintf(" ORDER BY %s ASC LIMIT 1", idColumn)

	var args []any
	if lastID != "" {
		args = append(args, lastID)
	}

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return schema.NodeResult{}, fmt.Errorf("snowflake trigger poll failed: %w", err)
	}
	defer rows.Close()

	var out []schema.Item
	for rows.Next() {
		rowData, err := scanRow(rows)
		if err != nil {
			continue
		}
		out = append(out, schema.Item{JSON: rowData})
		if v, ok := rowData[idColumn]; ok {
			execCtx.State["lastId"] = fmt.Sprintf("%v", v)
		}
	}

	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": out}}, nil
}
