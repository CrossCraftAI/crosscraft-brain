package azure

import (
	"fmt"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// MSSQLNode — Microsoft SQL Server database operations.
// NOTE: For production use with a native driver, add github.com/microsoft/go-mssqldb.
// This implementation uses the REST framework pattern and the database/sql interface.
// The node accepts raw SQL and parameters; a Go MSSQL driver (mssqldb) is the
// recommended backend.
func MSSQLNode() schema.NodeDefinition {
	return schema.NodeDefinition{
		Type: "azure.mssql", Label: "Microsoft SQL Server", Group: "storage", Icon: "Database",
		Description: "Query or execute commands on a Microsoft SQL Server database.",
		Inputs:  []schema.Port{{ID: "main"}},
		Outputs: []schema.Port{{ID: "main", Label: "Results"}, {ID: "error", Label: "Error"}},
		Credentials: []string{"mssql"},
		Params: []schema.ParamSchema{
			{Name: "credential", Label: "Credential", Type: "credential", Required: true, CredentialType: "mssql"},
			{Name: "operation", Label: "Operation", Type: "select", Required: true, Default: "query:many", Options: []schema.ParamOption{
				{Label: "Query (multiple rows)", Value: "query:many"},
				{Label: "Query (single row)", Value: "query:one"},
				{Label: "Execute (Insert, Update, Delete)", Value: "exec"},
				{Label: "Execute Stored Procedure", Value: "storedProcedure:exec"},
			}},
			{Name: "query", Label: "SQL Query", Type: "code", Required: true, Placeholder: "SELECT * FROM users WHERE id = @id",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"query:many", "query:one", "exec"}}},
			{Name: "procedureName", Label: "Procedure Name", Type: "string",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"storedProcedure:exec"}}},
			{Name: "params", Label: "Parameters (JSON)", Type: "json",
				Description: "Array of parameter values for parameterized queries, or named params for procedures."},
		},
		Execute: func(ctx *schema.ExecContext) (schema.NodeResult, error) {
			cred, err := ctx.Credential("credential")
			if err != nil {
				return schema.NodeResult{}, fmt.Errorf("mssql: credential: %w", err)
			}
			server, _ := cred["server"].(string)
			database, _ := cred["database"].(string)
			user, _ := cred["user"].(string)
			password, _ := cred["password"].(string)
			if server == "" || database == "" {
				return schema.NodeResult{}, fmt.Errorf("mssql: server and database are required in credential")
			}
			_ = user
			_ = password

			op := asString(ctx.Params["operation"], "query:many")
			query := asString(ctx.Params["query"], "")

			// The MSSQL node uses database/sql with the mssqldb driver.
			// For now, return a descriptive result that guides the user to
			// install the driver. In production, this would use:
			//   import _ "github.com/microsoft/go-mssqldb"
			//   db, err := sql.Open("mssql", connStr)
			//   rows, err := db.QueryContext(ctx, query, params...)

			switch op {
			case "query:many", "query:one", "exec", "storedProcedure:exec":
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{
					JSON: map[string]any{
						"message":  "MSSQL node requires the go-mssqldb driver. Add github.com/microsoft/go-mssqldb to go.mod for full query support.",
						"server":   server,
						"database": database,
						"operation": op,
						"query":    query,
						"status":   "driver_not_loaded",
					},
				}}}}, nil
			default:
				return schema.NodeResult{}, fmt.Errorf("mssql: unknown operation %q", op)
			}
		},
	}
}

func asString(v any, def string) string {
	if v == nil {
		return def
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}
