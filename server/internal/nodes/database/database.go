package database

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
	"github.com/jackc/pgx/v5"
)

// PostgresNode returns the definition for the PostgreSQL node.
func PostgresNode() schema.NodeDefinition {
	return schema.NodeDefinition{
		Type:        "database.postgres",
		Label:       "PostgreSQL",
		Description: "Query or execute commands on a PostgreSQL database.",
		Group:       "integration",
		Icon:        "Database",
		Inputs:      []schema.Port{{ID: "main"}},
		Outputs:     []schema.Port{{ID: "main", Label: "Results"}, {ID: "error", Label: "Error"}},
		Credentials: []string{"postgres"},
		Params: []schema.ParamSchema{
			{Name: "credential", Label: "Credential", Type: "credential", Required: true, CredentialType: "postgres"},
			{Name: "operation", Label: "Operation", Type: "select", Required: true, Default: "query:many", Options: []schema.ParamOption{
				{Label: "Query (multiple rows)", Value: "query:many"},
				{Label: "Query (single row)", Value: "query:one"},
				{Label: "Execute (Insert, Update, Delete)", Value: "exec"},
			}},
			{Name: "query", Label: "SQL Query", Type: "code:sql", Required: true},
			{Name: "params", Label: "Query Parameters", Type: "json", Description: "An array of values for parameterized queries (e.g., using $1, $2)."},
		},
		Execute: executePostgres,
	}
}

// Nodes returns the database node pack.
func Nodes() []schema.NodeDefinition {
	return []schema.NodeDefinition{PostgresNode()}
}

func resolvePostgresDSN(ctx *schema.ExecContext) (string, error) {
	if ctx == nil {
		return "", fmt.Errorf("postgres: missing execution context")
	}
	if ctx.Credential != nil {
		cred, err := ctx.Credential("credential")
		if err != nil {
			return "", fmt.Errorf("postgres: failed to get credentials: %w", err)
		}
		if len(cred) > 0 {
			if dsn, ok := cred["dsn"].(string); ok && dsn != "" {
				return dsn, nil
			}
			user, _ := cred["user"].(string)
			password, _ := cred["password"].(string)
			host, _ := cred["host"].(string)
			port, _ := cred["port"].(string)
			dbname, _ := cred["dbname"].(string)
			if host != "" && dbname != "" {
				if port == "" {
					port = "5432"
				}
				u := url.URL{
					Scheme: "postgres",
					User:   url.UserPassword(user, password),
					Host:   fmt.Sprintf("%s:%s", host, port),
					Path:   dbname,
				}
				return u.String(), nil
			}
		}
	}
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		return dsn, nil
	}
	return "", fmt.Errorf("postgres: no credential or DATABASE_URL configured")
}

// executePostgres is the execution function for the PostgreSQL node.
func executePostgres(ctx *schema.ExecContext) (schema.NodeResult, error) {
	// 1. Resolve a DSN from the node credential or the app's configured database URL.
	dsn, err := resolvePostgresDSN(ctx)
	if err != nil {
		return schema.NodeResult{}, err
	}

	// 2. Connect to the database with a timeout.
	queryCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := pgx.Connect(queryCtx, dsn)
	if err != nil {
		return schema.NodeResult{}, fmt.Errorf("postgres: unable to connect to database: %w", err)
	}
	defer conn.Close(context.Background())

	// 3. Get query and parameters
	query, _ := ctx.Params["query"].(string)
	rawParams := ctx.Params["params"]
	queryParams, _ := rawParams.([]any) // ok to be nil; pgx handles zero params

	operation, _ := ctx.Params["operation"].(string)
	out := make([]schema.Item, 0)

	// 4. Execute based on operation
	switch operation {
	case "query:many":
		rows, err := conn.Query(queryCtx, query, queryParams...)
		if err != nil {
			return schema.NodeResult{}, fmt.Errorf("postgres query failed: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			values, err := rows.Values()
			if err != nil {
				return schema.NodeResult{}, fmt.Errorf("postgres failed to read row values: %w", err)
			}
			fieldDescs := rows.FieldDescriptions()
			rowData := make(map[string]any)
			for i, fd := range fieldDescs {
				rowData[string(fd.Name)] = values[i]
			}
			out = append(out, schema.Item{JSON: rowData})
		}
		if err := rows.Err(); err != nil {
			return schema.NodeResult{}, fmt.Errorf("postgres rows error: %w", err)
		}

	case "query:one":
		rows, err := conn.Query(queryCtx, query, queryParams...)
		if err != nil {
			return schema.NodeResult{}, fmt.Errorf("postgres query failed: %w", err)
		}
		if rows.Next() {
			values, err := rows.Values()
			if err != nil {
				rows.Close()
				return schema.NodeResult{}, fmt.Errorf("postgres failed to read row values: %w", err)
			}
			fieldDescs := rows.FieldDescriptions()
			rowData := make(map[string]any)
			for i, fd := range fieldDescs {
				rowData[string(fd.Name)] = values[i]
			}
			out = append(out, schema.Item{JSON: rowData})
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return schema.NodeResult{}, fmt.Errorf("postgres rows error: %w", err)
		}

	case "exec":
		cmdTag, err := conn.Exec(queryCtx, query, queryParams...)
		if err != nil {
			return schema.NodeResult{}, fmt.Errorf("postgres exec failed: %w", err)
		}
		out = append(out, schema.Item{JSON: map[string]any{
			"status":        "success",
			"command":       cmdTag.String(),
			"rows_affected": cmdTag.RowsAffected(),
		}})

	default:
		return schema.NodeResult{}, fmt.Errorf("postgres: unknown operation %q", operation)
	}

	// 5. Return results
	return schema.NodeResult{
		Outputs: map[string][]schema.Item{"main": out},
	}, nil
}
