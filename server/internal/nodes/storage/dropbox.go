package storage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

var dropboxHTTPClient = &http.Client{Timeout: 60 * time.Second}

// DropboxNode returns the definition for the Dropbox node.
func DropboxNode() schema.NodeDefinition {
	return schema.NodeDefinition{
		Type:        "storage.dropbox",
		Label:       "Dropbox",
		Description: "Manage files and folders in Dropbox.",
		Group:       "integration",
		Icon:        "HardDrive",
		Inputs:      []schema.Port{{ID: "main"}},
		Outputs:     []schema.Port{{ID: "main", Label: "Results"}, {ID: "error", Label: "Error"}},
		Credentials: []string{"dropboxApi"},
		Params: []schema.ParamSchema{
			{Name: "credential", Label: "Credential", Type: "credential", Required: true, CredentialType: "dropboxApi"},
			{Name: "operation", Label: "Operation", Type: "select", Required: true, Default: "list", Options: []schema.ParamOption{
				{Label: "List Files", Value: "list"},
				{Label: "Upload File", Value: "upload"},
				{Label: "Download File", Value: "download"},
				{Label: "Delete", Value: "delete"},
				{Label: "Move", Value: "move"},
				{Label: "Copy", Value: "copy"},
				{Label: "Share", Value: "share"},
				{Label: "Trigger: New File", Value: "trigger:newFile"},
			}},
			{Name: "path", Label: "Path", Type: "string", Required: true, Placeholder: "/folder/file.txt",
				Description: "Dropbox file/folder path. Use empty string for root.",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"list", "download", "delete", "share", "trigger:newFile"}}},
			{Name: "toPath", Label: "Destination Path", Type: "string", Required: true,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"move", "copy"}}},
			{Name: "uploadPath", Label: "Upload Path", Type: "string", Required: true, Placeholder: "/folder/file.txt",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"upload"}}},
			{Name: "content", Label: "File Content (Base64 or text)", Type: "string",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"upload"}}},
			{Name: "recursive", Label: "Recursive List", Type: "boolean", Default: false,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"list"}}},
			{Name: "limit", Label: "Max Results", Type: "number", Default: 100,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"list"}}},
		},
		Execute: executeDropbox,
	}
}

// dropboxCall makes an HTTP request to the Dropbox API.
// For content-download endpoints, the response is on https://content.dropboxapi.com.
func dropboxCall(token, _ /*method*/, endpoint string, body any, extraHeaders map[string]string) ([]byte, int, error) {
	// Determine base URL: content endpoints use content.dropboxapi.com
	baseURL := "https://api.dropboxapi.com/2"
	if strings.HasPrefix(endpoint, "/files/download") || strings.HasPrefix(endpoint, "/files/get_thumbnail") {
		baseURL = "https://content.dropboxapi.com/2"
	}

	u := baseURL + endpoint
	var reqBody io.Reader
	var bodyBytes []byte
	if body != nil {
		bodyBytes, _ = json.Marshal(body)
		reqBody = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequest(http.MethodPost, u, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("dropbox: request error: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}

	resp, err := dropboxHTTPClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("dropbox: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, resp.StatusCode, fmt.Errorf("dropbox: %d %s", resp.StatusCode, truncateStrMax(string(raw), 400))
	}
	return raw, resp.StatusCode, nil
}

func truncateStrMax(s string, n int) string {
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}

// executeDropbox is the execution function for the Dropbox node.
func executeDropbox(ctx *schema.ExecContext) (schema.NodeResult, error) {
	cred, err := ctx.Credential("credential")
	if err != nil {
		return schema.NodeResult{}, fmt.Errorf("dropbox: failed to get credentials: %w", err)
	}
	token, _ := cred["accessToken"].(string)
	if token == "" {
		return schema.NodeResult{}, fmt.Errorf("dropbox: access token is required")
	}

	operation, _ := ctx.Params["operation"].(string)

	switch operation {
	case "list":
		return dropboxList(ctx, token)
	case "upload":
		return dropboxUpload(ctx, token)
	case "download":
		return dropboxDownload(ctx, token)
	case "delete":
		return dropboxDelete(ctx, token)
	case "move":
		return dropboxMove(ctx, token)
	case "copy":
		return dropboxCopy(ctx, token)
	case "share":
		return dropboxShare(ctx, token)
	case "trigger:newFile":
		return dropboxTriggerNewFile(ctx, token)
	default:
		return schema.NodeResult{}, fmt.Errorf("dropbox: unknown operation %q", operation)
	}
}

func dropboxList(ctx *schema.ExecContext, token string) (schema.NodeResult, error) {
	path, _ := ctx.Params["path"].(string)
	recursive, _ := ctx.Params["recursive"].(bool)
	limit := 100
	if v, ok := ctx.Params["limit"].(int); ok && v > 0 {
		limit = v
	} else if v, ok := ctx.Params["limit"].(float64); ok && v > 0 {
		limit = int(v)
	}

	reqBody := map[string]any{
		"path":                                path,
		"recursive":                           recursive,
		"include_media_info":                  false,
		"include_deleted":                     false,
		"include_has_explicit_shared_members": false,
		"limit": limit,
	}

	raw, _, err := dropboxCall(token, "POST", "/files/list_folder", reqBody, nil)
	if err != nil {
		return schema.NodeResult{}, err
	}

	var result map[string]any
	if json.Unmarshal(raw, &result) != nil {
		return schema.NodeResult{}, fmt.Errorf("dropbox: failed to parse response")
	}

	var out []schema.Item
	if entries, ok := result["entries"].([]any); ok {
		for _, e := range entries {
			if m, ok := e.(map[string]any); ok {
				out = append(out, schema.Item{JSON: m})
			}
		}
	}
	if out == nil {
		out = []schema.Item{}
	}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": out}}, nil
}

func dropboxUpload(ctx *schema.ExecContext, token string) (schema.NodeResult, error) {
	uploadPath, _ := ctx.Params["uploadPath"].(string)
	content, _ := ctx.Params["content"].(string)
	if uploadPath == "" {
		return schema.NodeResult{}, fmt.Errorf("dropbox: upload path is required")
	}

	// Dropbox content upload uses https://content.dropboxapi.com/2/files/upload
	// with a Dropbox-API-Arg header containing JSON metadata
	apiArg, _ := json.Marshal(map[string]any{
		"path":       uploadPath,
		"mode":       "add",
		"autorename": true,
		"mute":       false,
	})

	extraHeaders := map[string]string{
		"Dropbox-API-Arg": string(apiArg),
		"Content-Type":    "application/octet-stream",
	}

	raw, status, err := dropboxCallRaw(token, "https://content.dropboxapi.com/2/files/upload", []byte(content), extraHeaders)
	if err != nil {
		return schema.NodeResult{}, err
	}

	var result map[string]any
	if status >= 200 && status < 300 && len(raw) > 0 {
		json.Unmarshal(raw, &result)
	}
	if result == nil {
		result = map[string]any{"status": "uploaded", "path": uploadPath}
	}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{JSON: result}}}}, nil
}

// dropboxCallRaw makes a raw HTTP request (for file content uploads/downloads).
func dropboxCallRaw(token, url string, body []byte, headers map[string]string) ([]byte, int, error) {
	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}

	req, err := http.NewRequest(http.MethodPost, url, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("dropbox: request error: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := dropboxHTTPClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("dropbox: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, resp.StatusCode, fmt.Errorf("dropbox: %d %s", resp.StatusCode, truncateStrMax(string(raw), 400))
	}
	return raw, resp.StatusCode, nil
}

func dropboxDownload(ctx *schema.ExecContext, token string) (schema.NodeResult, error) {
	path, _ := ctx.Params["path"].(string)
	if path == "" {
		return schema.NodeResult{}, fmt.Errorf("dropbox: path is required")
	}

	apiArg, _ := json.Marshal(map[string]string{"path": path})
	extraHeaders := map[string]string{"Dropbox-API-Arg": string(apiArg)}

	raw, _, err := dropboxCallRaw(token, "https://content.dropboxapi.com/2/files/download", nil, extraHeaders)
	if err != nil {
		return schema.NodeResult{}, err
	}

	// Store as binary data reference
	item := schema.Item{
		JSON: map[string]any{"path": path, "size": len(raw), "downloaded": true},
		Binary: map[string]schema.BinaryRef{
			"data": {Data: string(raw), MimeType: "application/octet-stream", FileName: path[strings.LastIndex(path, "/")+1:]},
		},
	}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {item}}}, nil
}

func dropboxDelete(ctx *schema.ExecContext, token string) (schema.NodeResult, error) {
	path, _ := ctx.Params["path"].(string)
	if path == "" {
		return schema.NodeResult{}, fmt.Errorf("dropbox: path is required")
	}

	raw, _, err := dropboxCall(token, "POST", "/files/delete_v2", map[string]string{"path": path}, nil)
	if err != nil {
		return schema.NodeResult{}, err
	}

	var result map[string]any
	json.Unmarshal(raw, &result)
	if result == nil {
		result = map[string]any{"deleted": true, "path": path}
	}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{JSON: result}}}}, nil
}

func dropboxMove(ctx *schema.ExecContext, token string) (schema.NodeResult, error) {
	fromPath, _ := ctx.Params["path"].(string)
	toPath, _ := ctx.Params["toPath"].(string)
	if fromPath == "" || toPath == "" {
		return schema.NodeResult{}, fmt.Errorf("dropbox: source and destination paths are required")
	}

	raw, _, err := dropboxCall(token, "POST", "/files/move_v2", map[string]string{
		"from_path": fromPath,
		"to_path":   toPath,
	}, nil)
	if err != nil {
		return schema.NodeResult{}, err
	}

	var result map[string]any
	json.Unmarshal(raw, &result)
	if result == nil {
		result = map[string]any{"moved": true, "from": fromPath, "to": toPath}
	}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{JSON: result}}}}, nil
}

func dropboxCopy(ctx *schema.ExecContext, token string) (schema.NodeResult, error) {
	fromPath, _ := ctx.Params["path"].(string)
	toPath, _ := ctx.Params["toPath"].(string)
	if fromPath == "" || toPath == "" {
		return schema.NodeResult{}, fmt.Errorf("dropbox: source and destination paths are required")
	}

	raw, _, err := dropboxCall(token, "POST", "/files/copy_v2", map[string]string{
		"from_path": fromPath,
		"to_path":   toPath,
	}, nil)
	if err != nil {
		return schema.NodeResult{}, err
	}

	var result map[string]any
	json.Unmarshal(raw, &result)
	if result == nil {
		result = map[string]any{"copied": true, "from": fromPath, "to": toPath}
	}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{JSON: result}}}}, nil
}

func dropboxShare(ctx *schema.ExecContext, token string) (schema.NodeResult, error) {
	path, _ := ctx.Params["path"].(string)
	if path == "" {
		return schema.NodeResult{}, fmt.Errorf("dropbox: path is required")
	}

	raw, _, err := dropboxCall(token, "POST", "/sharing/create_shared_link_with_settings", map[string]any{
		"path":     path,
		"settings": map[string]any{"requested_visibility": "public"},
	}, nil)
	if err != nil {
		return schema.NodeResult{}, err
	}

	var result map[string]any
	json.Unmarshal(raw, &result)
	if result == nil {
		result = map[string]any{"shared": false, "path": path}
	}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{JSON: result}}}}, nil
}

func dropboxTriggerNewFile(ctx *schema.ExecContext, token string) (schema.NodeResult, error) {
	path, _ := ctx.Params["path"].(string)
	if path == "" {
		path = ""
	}

	// Use the list_folder/continue API with a cursor from persistent state
	// to poll for changes
	lastCursor, _ := ctx.State["cursor"].(string)

	var listFunc string
	var listBody map[string]any
	if lastCursor != "" {
		listFunc = "/files/list_folder/continue"
		listBody = map[string]any{"cursor": lastCursor}
	} else {
		listFunc = "/files/list_folder"
		listBody = map[string]any{
			"path":      path,
			"recursive": true,
			"limit":     10,
		}
	}

	raw, _, err := dropboxCall(token, "POST", listFunc, listBody, nil)
	if err != nil {
		return schema.NodeResult{}, fmt.Errorf("dropbox trigger poll failed: %w", err)
	}

	var result map[string]any
	if json.Unmarshal(raw, &result) != nil {
		return schema.NodeResult{}, fmt.Errorf("dropbox: failed to parse response")
	}

	// Update cursor for next poll
	if cursor, ok := result["cursor"].(string); ok {
		ctx.State["cursor"] = cursor
	}

	var out []schema.Item
	if entries, ok := result["entries"].([]any); ok {
		for _, e := range entries {
			if m, ok := e.(map[string]any); ok {
				out = append(out, schema.Item{JSON: m})
			}
		}
	}
	if out == nil {
		out = []schema.Item{}
	}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": out}}, nil
}
