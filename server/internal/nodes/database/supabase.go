package database

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

var supabaseHTTPClient = &http.Client{Timeout: 30 * time.Second}

// SupabaseNode returns the definition for the Supabase node.
func SupabaseNode() schema.NodeDefinition {
	return schema.NodeDefinition{
		Type:        "database.supabase",
		Label:       "Supabase",
		Description: "Query and manage data in Supabase (PostgreSQL REST API).",
		Group:       "storage",
		Icon:        "Database",
		Inputs:      []schema.Port{{ID: "main"}},
		Outputs:     []schema.Port{{ID: "main", Label: "Results"}, {ID: "error", Label: "Error"}},
		Credentials: []string{"supabaseApi"},
		Params: []schema.ParamSchema{
			{Name: "credential", Label: "Credential", Type: "credential", Required: true, CredentialType: "supabaseApi"},
			{Name: "operation", Label: "Operation", Type: "select", Required: true, Default: "select", Options: []schema.ParamOption{
				{Label: "Select (query rows)", Value: "select"},
				{Label: "Insert", Value: "insert"},
				{Label: "Update", Value: "update"},
				{Label: "Delete", Value: "delete"},
				{Label: "RPC (stored procedure)", Value: "rpc"},
				{Label: "Trigger: New Record", Value: "trigger:newRecord"},
			}},
			{Name: "table", Label: "Table Name", Type: "string", Required: true,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"select", "insert", "update", "delete", "trigger:newRecord"}}},
			{Name: "rpcName", Label: "RPC Function Name", Type: "string", Required: true,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"rpc"}}},
			{Name: "query", Label: "Query Parameters (URL query string)", Type: "json",
				Description: "URL query filters, e.g. {\"column\": \"eq.value\", \"select\": \"col1,col2\", \"limit\": 10, \"order\": \"id.desc\"}",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"select", "update", "delete"}}},
			{Name: "body", Label: "Body (JSON)", Type: "json",
				Description: "Row data for insert/update, or RPC params.",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"insert", "update", "rpc"}}},
			{Name: "idColumn", Label: "ID Column", Type: "string", Default: "id",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"trigger:newRecord"}}},
		},
		Execute: executeSupabase,
	}
}

// resolveSupabaseConfig extracts the project URL and API key from the credential.
func resolveSupabaseConfig(ctx *schema.ExecContext) (string, string, error) {
	if ctx.Credential != nil {
		cred, err := ctx.Credential("credential")
		if err != nil {
			return "", "", fmt.Errorf("supabase: failed to get credentials: %w", err)
		}
		if len(cred) > 0 {
			baseURL, _ := cred["url"].(string)
			apiKey, _ := cred["accessToken"].(string)
			if baseURL == "" || apiKey == "" {
				return "", "", fmt.Errorf("supabase: project URL and access token are required")
			}
			baseURL = strings.TrimRight(baseURL, "/")
			return baseURL, apiKey, nil
		}
	}
	return "", "", fmt.Errorf("supabase: no credential configured")
}

// supabaseCall makes an HTTP request to the Supabase REST API.
func supabaseCall(method, baseURL, apiKey, path string, body map[string]any, queryParams map[string]any) ([]schema.Item, error) {
	u := baseURL + path
	if len(queryParams) > 0 {
		q := url.Values{}
		for k, v := range queryParams {
			q.Set(k, fmt.Sprintf("%v", v))
		}
		u += "?" + q.Encode()
	}

	var reqBody io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, u, reqBody)
	if err != nil {
		return nil, fmt.Errorf("supabase: request error: %w", err)
	}
	req.Header.Set("apikey", apiKey)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	// Prefer returning representation after mutation
	if method == http.MethodPost || method == http.MethodPatch {
		req.Header.Set("Prefer", "return=representation")
	}

	resp, err := supabaseHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("supabase: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("supabase: %d %s", resp.StatusCode, truncateStr(string(raw), 400))
	}

	if len(bytes.TrimSpace(raw)) == 0 {
		return []schema.Item{{JSON: map[string]any{"success": true}}}, nil
	}

	// Try to parse as array first, then object
	var root any
	if json.Unmarshal(raw, &root) != nil {
		return []schema.Item{{JSON: map[string]any{"raw": string(raw)}}}, nil
	}

	if arr, ok := root.([]any); ok {
		out := make([]schema.Item, 0, len(arr))
		for _, e := range arr {
			if m, ok := e.(map[string]any); ok {
				out = append(out, schema.Item{JSON: m})
			}
		}
		return out, nil
	}
	if m, ok := root.(map[string]any); ok {
		return []schema.Item{{JSON: m}}, nil
	}
	return []schema.Item{{JSON: map[string]any{"success": true}}}, nil
}

func truncateStr(s string, n int) string {
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}

// executeSupabase is the execution function for the Supabase node.
func executeSupabase(ctx *schema.ExecContext) (schema.NodeResult, error) {
	baseURL, apiKey, err := resolveSupabaseConfig(ctx)
	if err != nil {
		return schema.NodeResult{}, err
	}

	operation, _ := ctx.Params["operation"].(string)

	switch operation {
	case "select":
		return supabaseSelect(ctx, baseURL, apiKey)
	case "insert":
		return supabaseInsert(ctx, baseURL, apiKey)
	case "update":
		return supabaseUpdate(ctx, baseURL, apiKey)
	case "delete":
		return supabaseDelete(ctx, baseURL, apiKey)
	case "rpc":
		return supabaseRPC(ctx, baseURL, apiKey)
	case "trigger:newRecord":
		return supabaseTriggerNewRecord(ctx, baseURL, apiKey)
	default:
		return schema.NodeResult{}, fmt.Errorf("supabase: unknown operation %q", operation)
	}
}

func supabaseSelect(ctx *schema.ExecContext, baseURL, apiKey string) (schema.NodeResult, error) {
	table, _ := ctx.Params["table"].(string)
	if table == "" {
		return schema.NodeResult{}, fmt.Errorf("supabase: table name is required")
	}

	queryParams := map[string]any{}
	if q := asObjectValue(ctx.RawParam("query")); q != nil {
		for k, v := range q {
			queryParams[k] = v
		}
	}

	items, err := supabaseCall("GET", baseURL, apiKey, "/rest/v1/"+table, nil, queryParams)
	if err != nil {
		return schema.NodeResult{}, err
	}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": items}}, nil
}

func supabaseInsert(ctx *schema.ExecContext, baseURL, apiKey string) (schema.NodeResult, error) {
	table, _ := ctx.Params["table"].(string)
	if table == "" {
		return schema.NodeResult{}, fmt.Errorf("supabase: table name is required")
	}

	body := asObjectValue(ctx.RawParam("body"))
	if body == nil {
		return schema.NodeResult{}, fmt.Errorf("supabase: body is required for insert")
	}

	items, err := supabaseCall("POST", baseURL, apiKey, "/rest/v1/"+table, body, nil)
	if err != nil {
		return schema.NodeResult{}, err
	}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": items}}, nil
}

func supabaseUpdate(ctx *schema.ExecContext, baseURL, apiKey string) (schema.NodeResult, error) {
	table, _ := ctx.Params["table"].(string)
	if table == "" {
		return schema.NodeResult{}, fmt.Errorf("supabase: table name is required")
	}

	body := asObjectValue(ctx.RawParam("body"))
	if body == nil {
		return schema.NodeResult{}, fmt.Errorf("supabase: body is required for update")
	}

	queryParams := map[string]any{}
	if q := asObjectValue(ctx.RawParam("query")); q != nil {
		for k, v := range q {
			queryParams[k] = v
		}
	}

	// Body values are the SET clauses; query params are the WHERE filters.
	// Keep both separate — Supabase uses query params for filtering.

	items, err := supabaseCall("PATCH", baseURL, apiKey, "/rest/v1/"+table, body, queryParams)
	if err != nil {
		return schema.NodeResult{}, err
	}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": items}}, nil
}

func supabaseDelete(ctx *schema.ExecContext, baseURL, apiKey string) (schema.NodeResult, error) {
	table, _ := ctx.Params["table"].(string)
	if table == "" {
		return schema.NodeResult{}, fmt.Errorf("supabase: table name is required")
	}

	queryParams := map[string]any{}
	if q := asObjectValue(ctx.RawParam("query")); q != nil {
		for k, v := range q {
			queryParams[k] = v
		}
	}

	items, err := supabaseCall("DELETE", baseURL, apiKey, "/rest/v1/"+table, nil, queryParams)
	if err != nil {
		return schema.NodeResult{}, err
	}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": items}}, nil
}

func supabaseRPC(ctx *schema.ExecContext, baseURL, apiKey string) (schema.NodeResult, error) {
	rpcName, _ := ctx.Params["rpcName"].(string)
	if rpcName == "" {
		return schema.NodeResult{}, fmt.Errorf("supabase: RPC function name is required")
	}
	body := asObjectValue(ctx.RawParam("body"))

	items, err := supabaseCall("POST", baseURL, apiKey, "/rest/v1/rpc/"+rpcName, body, nil)
	if err != nil {
		return schema.NodeResult{}, err
	}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": items}}, nil
}

func supabaseTriggerNewRecord(ctx *schema.ExecContext, baseURL, apiKey string) (schema.NodeResult, error) {
	table, _ := ctx.Params["table"].(string)
	if table == "" {
		return schema.NodeResult{}, fmt.Errorf("supabase trigger: table name is required")
	}
	idColumn, _ := ctx.Params["idColumn"].(string)
	if idColumn == "" {
		idColumn = "id"
	}

	lastID, _ := ctx.State["lastId"].(string)
	queryParams := map[string]any{
		"order": idColumn + ".desc",
		"limit": "1",
	}
	if lastID != "" {
		queryParams[idColumn] = "gt." + lastID
		queryParams["order"] = idColumn + ".asc"
	}

	items, err := supabaseCall("GET", baseURL, apiKey, "/rest/v1/"+table, nil, queryParams)
	if err != nil {
		return schema.NodeResult{}, err
	}

	// Update cursor state
	for _, item := range items {
		if v, ok := item.JSON[idColumn]; ok {
			ctx.State["lastId"] = fmt.Sprintf("%v", v)
			break
		}
	}

	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": items}}, nil
}

