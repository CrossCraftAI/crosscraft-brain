package azure

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/rest"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

// CosmosDB — Azure Cosmos DB (Core SQL API) via REST with Master Key auth.
func CosmosDB(base string) rest.Node {
	body := jp("body", "Body (JSON)")
	dbID := sp("databaseId", "Database ID", true)
	containerID := sp("containerId", "Container ID", true)
	itemID := sp("itemId", "Item ID", true)
	partitionKey := schema.ParamSchema{Name: "partitionKey", Label: "Partition key value", Type: "string"}
	sqlQuery := schema.ParamSchema{Name: "query", Label: "SQL Query", Type: "code"}
	maxResults := ip("maxResults", "Max results", 100)
	continueToken := schema.ParamSchema{Name: "continuationToken", Label: "Continuation token", Type: "string"}

	return rest.Node{
		Type: "azure.cosmos", Label: "Azure Cosmos DB", Group: "integration", Icon: "Database",
		Description:  "Manage Cosmos DB databases, containers, items, and queries.",
		BaseURL:      base,
		BaseURLParam: "baseUrl",
		CredType:     "azureCosmos",
		Auth:         rest.Auth{Kind: "none"}, // Master Key auth handled via custom sign
		Ops: []rest.Op{
			// Databases
			{Resource: "database", Name: "list", Label: "List Databases", Method: "GET",
				Path: "/dbs", ItemsPath: "Databases",
				Params: []schema.ParamSchema{maxResults}},
			{Resource: "database", Name: "get", Label: "Get Database", Method: "GET",
				Path: "/dbs/{databaseId}",
				Params: []schema.ParamSchema{dbID}},
			{Resource: "database", Name: "create", Label: "Create Database", Method: "POST",
				Path: "/dbs", BodyParam: "body",
				Params: []schema.ParamSchema{body}},
			{Resource: "database", Name: "delete", Label: "Delete Database", Method: "DELETE",
				Path: "/dbs/{databaseId}",
				Params: []schema.ParamSchema{dbID}},
			// Containers
			{Resource: "container", Name: "list", Label: "List Containers", Method: "GET",
				Path: "/dbs/{databaseId}/colls",
				Params: []schema.ParamSchema{dbID}},
			{Resource: "container", Name: "create", Label: "Create Container", Method: "POST",
				Path: "/dbs/{databaseId}/colls", BodyParam: "body",
				Params: []schema.ParamSchema{dbID, body}},
			{Resource: "container", Name: "delete", Label: "Delete Container", Method: "DELETE",
				Path: "/dbs/{databaseId}/colls/{containerId}",
				Params: []schema.ParamSchema{dbID, containerID}},
			// Documents (items)
			{Resource: "item", Name: "get", Label: "Get Item", Method: "GET",
				Path: "/dbs/{databaseId}/colls/{containerId}/docs/{itemId}",
				Params: []schema.ParamSchema{dbID, containerID, itemID, partitionKey}},
			{Resource: "item", Name: "create", Label: "Create Item", Method: "POST",
				Path: "/dbs/{databaseId}/colls/{containerId}/docs", BodyParam: "body",
				Params: []schema.ParamSchema{dbID, containerID, body}},
			{Resource: "item", Name: "update", Label: "Update Item", Method: "PUT",
				Path: "/dbs/{databaseId}/colls/{containerId}/docs/{itemId}", BodyParam: "body",
				Params: []schema.ParamSchema{dbID, containerID, itemID, body}},
			{Resource: "item", Name: "delete", Label: "Delete Item", Method: "DELETE",
				Path: "/dbs/{databaseId}/colls/{containerId}/docs/{itemId}",
				Params: []schema.ParamSchema{dbID, containerID, itemID, partitionKey}},
			// Query
			{Resource: "item", Name: "query", Label: "Query Items (SQL)", Method: "POST",
				Path: "/dbs/{databaseId}/colls/{containerId}/docs", BodyParam: "body",
				Params: []schema.ParamSchema{dbID, containerID, sqlQuery, maxResults, continueToken, body}},
		},
	}
}

// cosmosSignRequest signs a Cosmos DB REST API request with the master key.
func cosmosSignRequest(req *http.Request, key string) error {
	decoded, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return fmt.Errorf("cosmos: decode key: %w", err)
	}

	// Build string to sign per Azure Cosmos DB REST spec
	verb := strings.ToLower(req.Method)
	resourceType := ""
	resourceID := req.URL.Path
	if strings.Contains(resourceID, "/docs") {
		resourceType = "docs"
	} else if strings.Contains(resourceID, "/colls") && !strings.Contains(resourceID, "/docs") {
		resourceType = "colls"
	} else if strings.Contains(resourceID, "/dbs") && !strings.Contains(resourceID, "/colls") {
		resourceType = "dbs"
	}

	date := req.Header.Get("x-ms-date")
	if date == "" {
		date = time.Now().UTC().Format(http.TimeFormat)
		req.Header.Set("x-ms-date", date)
	}

	stringToSign := strings.ToLower(verb) + "\n" +
		resourceType + "\n" +
		resourceID + "\n" +
		date + "\n" +
		"\n" // empty for master key token

	mac := hmac.New(sha256.New, decoded)
	mac.Write([]byte(stringToSign))
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	token := url.QueryEscape("type=master&ver=1.0&sig=" + sig)
	req.Header.Set("Authorization", token)
	req.Header.Set("x-ms-version", "2018-12-31")
	return nil
}

// cosmosDo executes a signed Cosmos DB request.
func cosmosDo(method, baseURL, path, key string, body io.Reader) ([]schema.Item, error) {
	req, err := http.NewRequest(method, strings.TrimRight(baseURL, "/")+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-ms-date", time.Now().UTC().Format(http.TimeFormat))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	if err := cosmosSignRequest(req, key); err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cosmos: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("cosmos: %d %s", resp.StatusCode, truncateS(string(raw), 400))
	}

	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		return []schema.Item{{JSON: map[string]any{"raw": string(raw)}}}, nil
	}
	return cosmosAnyToItems(result), nil
}

func cosmosAnyToItems(v any) []schema.Item {
	if m, ok := v.(map[string]any); ok {
		// Check for Documents array (query results)
		if docs, ok := m["Documents"].([]any); ok {
			items := make([]schema.Item, 0, len(docs))
			for _, d := range docs {
				if dm, ok := d.(map[string]any); ok {
					items = append(items, schema.Item{JSON: dm})
				}
			}
			return items
		}
		// Check for Databases
		if dbs, ok := m["Databases"].([]any); ok {
			items := make([]schema.Item, 0, len(dbs))
			for _, d := range dbs {
				if dm, ok := d.(map[string]any); ok {
					items = append(items, schema.Item{JSON: dm})
				}
			}
			return items
		}
		return []schema.Item{{JSON: m}}
	}
	if arr, ok := v.([]any); ok {
		items := make([]schema.Item, 0, len(arr))
		for _, e := range arr {
			if m, ok := e.(map[string]any); ok {
				items = append(items, schema.Item{JSON: m})
			}
		}
		return items
	}
	return []schema.Item{{JSON: map[string]any{"value": v}}}
}

func truncateS(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}
