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

var boxHTTPClient = &http.Client{Timeout: 60 * time.Second}

// BoxNode returns the definition for the Box node.
func BoxNode() schema.NodeDefinition {
	return schema.NodeDefinition{
		Type:        "storage.box",
		Label:       "Box",
		Description: "Manage files and folders in Box.",
		Group:       "integration",
		Icon:        "HardDrive",
		Inputs:      []schema.Port{{ID: "main"}},
		Outputs:     []schema.Port{{ID: "main", Label: "Results"}, {ID: "error", Label: "Error"}},
		Credentials: []string{"boxApi"},
		Params: []schema.ParamSchema{
			{Name: "credential", Label: "Credential", Type: "credential", Required: true, CredentialType: "boxApi"},
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
			{Name: "folderId", Label: "Folder ID", Type: "string", Required: true, Default: "0",
				Description: "Box folder ID. Use '0' for the root folder.",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"list", "upload", "trigger:newFile"}}},
			{Name: "fileId", Label: "File ID", Type: "string", Required: true,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"download", "delete", "move", "copy", "share"}}},
			{Name: "toFolderId", Label: "Destination Folder ID", Type: "string", Required: true,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"move", "copy"}}},
			{Name: "fileName", Label: "File Name", Type: "string", Required: true,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"upload"}}},
			{Name: "content", Label: "File Content (Base64 or text)", Type: "string",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"upload"}}},
			{Name: "limit", Label: "Max Results", Type: "number", Default: 100,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"list"}}},
		},
		Execute: executeBox,
	}
}

// boxCall makes an HTTP request to the Box API (v2.0).
func boxCall(token, method, endpoint string, body any) ([]byte, int, error) {
	baseURL := "https://api.box.com/2.0"
	// Upload API uses https://upload.box.com/api/2.0
	if strings.HasPrefix(endpoint, "/files/content") && method == http.MethodPost {
		baseURL = "https://upload.box.com/api/2.0"
	}

	u := baseURL + endpoint
	var reqBody io.Reader
	var bodyBytes []byte
	if body != nil {
		bodyBytes, _ = json.Marshal(body)
		reqBody = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequest(method, u, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("box: request error: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := boxHTTPClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("box: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, resp.StatusCode, fmt.Errorf("box: %d %s", resp.StatusCode, truncateStrMax(string(raw), 400))
	}
	return raw, resp.StatusCode, nil
}

// executeBox is the execution function for the Box node.
func executeBox(ctx *schema.ExecContext) (schema.NodeResult, error) {
	cred, err := ctx.Credential("credential")
	if err != nil {
		return schema.NodeResult{}, fmt.Errorf("box: failed to get credentials: %w", err)
	}
	token, _ := cred["accessToken"].(string)
	if token == "" {
		return schema.NodeResult{}, fmt.Errorf("box: access token is required")
	}

	operation, _ := ctx.Params["operation"].(string)

	switch operation {
	case "list":
		return boxList(ctx, token)
	case "upload":
		return boxUpload(ctx, token)
	case "download":
		return boxDownload(ctx, token)
	case "delete":
		return boxDelete(ctx, token)
	case "move":
		return boxMove(ctx, token)
	case "copy":
		return boxCopy(ctx, token)
	case "share":
		return boxShare(ctx, token)
	case "trigger:newFile":
		return boxTriggerNewFile(ctx, token)
	default:
		return schema.NodeResult{}, fmt.Errorf("box: unknown operation %q", operation)
	}
}

func boxList(ctx *schema.ExecContext, token string) (schema.NodeResult, error) {
	folderID, _ := ctx.Params["folderId"].(string)
	if folderID == "" {
		folderID = "0"
	}
	limit := 100
	if v, ok := ctx.Params["limit"].(int); ok && v > 0 {
		limit = v
	} else if v, ok := ctx.Params["limit"].(float64); ok && v > 0 {
		limit = int(v)
	}

	endpoint := fmt.Sprintf("/folders/%s/items?limit=%d&fields=id,name,type,size,modified_at,shared_link", folderID, limit)
	raw, _, err := boxCall(token, "GET", endpoint, nil)
	if err != nil {
		return schema.NodeResult{}, err
	}

	var result map[string]any
	if json.Unmarshal(raw, &result) != nil {
		return schema.NodeResult{}, fmt.Errorf("box: failed to parse response")
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

func boxUpload(ctx *schema.ExecContext, token string) (schema.NodeResult, error) {
	folderID, _ := ctx.Params["folderId"].(string)
	fileName, _ := ctx.Params["fileName"].(string)
	content, _ := ctx.Params["content"].(string)
	if folderID == "" {
		folderID = "0"
	}
	if fileName == "" {
		return schema.NodeResult{}, fmt.Errorf("box: file name is required")
	}

	// Box multipart upload attributes sent via the attributes header
	attrs := fmt.Sprintf(`{"name":"%s","parent":{"id":"%s"}}`, fileName, folderID)

	u := fmt.Sprintf("https://upload.box.com/api/2.0/files/content")
	var reqBody io.Reader
	if content != "" {
		reqBody = strings.NewReader(content)
	} else {
		reqBody = bytes.NewReader([]byte{})
	}

	req, err := http.NewRequest(http.MethodPost, u, reqBody)
	if err != nil {
		return schema.NodeResult{}, fmt.Errorf("box: upload request error: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
	req.Header.Set("attributes", attrs)

	resp, err := boxHTTPClient.Do(req)
	if err != nil {
		return schema.NodeResult{}, fmt.Errorf("box: upload error: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return schema.NodeResult{}, fmt.Errorf("box: upload failed: %d %s", resp.StatusCode, truncateStrMax(string(raw), 400))
	}

	var result map[string]any
	json.Unmarshal(raw, &result)
	if result == nil {
		result = map[string]any{"uploaded": true, "name": fileName, "parentId": folderID}
	}

	// Extract entries array if present
	var out []schema.Item
	if entries, ok := result["entries"].([]any); ok {
		for _, e := range entries {
			if m, ok := e.(map[string]any); ok {
				out = append(out, schema.Item{JSON: m})
			}
		}
	}
	if out == nil {
		out = []schema.Item{{JSON: result}}
	}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": out}}, nil
}

func boxDownload(ctx *schema.ExecContext, token string) (schema.NodeResult, error) {
	fileID, _ := ctx.Params["fileId"].(string)
	if fileID == "" {
		return schema.NodeResult{}, fmt.Errorf("box: file ID is required")
	}

	raw, _, err := boxCall(token, "GET", "/files/"+fileID+"/content", nil)
	if err != nil {
		return schema.NodeResult{}, err
	}

	item := schema.Item{
		JSON: map[string]any{"fileId": fileID, "size": len(raw), "downloaded": true},
		Binary: map[string]schema.BinaryRef{
			"data": {Data: string(raw), MimeType: "application/octet-stream", FileName: fileID},
		},
	}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {item}}}, nil
}

func boxDelete(ctx *schema.ExecContext, token string) (schema.NodeResult, error) {
	fileID, _ := ctx.Params["fileId"].(string)
	if fileID == "" {
		return schema.NodeResult{}, fmt.Errorf("box: file ID is required")
	}

	_, status, err := boxCall(token, "DELETE", "/files/"+fileID, nil)
	if err != nil && status != 404 {
		return schema.NodeResult{}, err
	}

	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{JSON: map[string]any{"deleted": true, "fileId": fileID}}}}}, nil
}

func boxMove(ctx *schema.ExecContext, token string) (schema.NodeResult, error) {
	fileID, _ := ctx.Params["fileId"].(string)
	toFolderID, _ := ctx.Params["toFolderId"].(string)
	if fileID == "" || toFolderID == "" {
		return schema.NodeResult{}, fmt.Errorf("box: file ID and destination folder ID are required")
	}

	body := map[string]any{
		"parent": map[string]string{"id": toFolderID},
	}

	raw, _, err := boxCall(token, "PUT", "/files/"+fileID, body)
	if err != nil {
		return schema.NodeResult{}, err
	}

	var result map[string]any
	json.Unmarshal(raw, &result)
	if result == nil {
		result = map[string]any{"moved": true, "fileId": fileID, "toFolderId": toFolderID}
	}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{JSON: result}}}}, nil
}

func boxCopy(ctx *schema.ExecContext, token string) (schema.NodeResult, error) {
	fileID, _ := ctx.Params["fileId"].(string)
	toFolderID, _ := ctx.Params["toFolderId"].(string)
	if fileID == "" || toFolderID == "" {
		return schema.NodeResult{}, fmt.Errorf("box: file ID and destination folder ID are required")
	}

	body := map[string]any{
		"parent": map[string]string{"id": toFolderID},
	}

	raw, _, err := boxCall(token, "POST", "/files/"+fileID+"/copy", body)
	if err != nil {
		return schema.NodeResult{}, err
	}

	var result map[string]any
	json.Unmarshal(raw, &result)
	if result == nil {
		result = map[string]any{"copied": true, "fileId": fileID, "toFolderId": toFolderID}
	}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{JSON: result}}}}, nil
}

func boxShare(ctx *schema.ExecContext, token string) (schema.NodeResult, error) {
	fileID, _ := ctx.Params["fileId"].(string)
	if fileID == "" {
		return schema.NodeResult{}, fmt.Errorf("box: file ID is required")
	}

	body := map[string]any{
		"shared_link": map[string]any{
			"access":     "open",
			"permissions": map[string]bool{"can_download": true},
		},
	}

	raw, _, err := boxCall(token, "PUT", "/files/"+fileID, body)
	if err != nil {
		return schema.NodeResult{}, err
	}

	var result map[string]any
	json.Unmarshal(raw, &result)
	if result == nil {
		result = map[string]any{"shared": false, "fileId": fileID}
	}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{JSON: result}}}}, nil
}

func boxTriggerNewFile(ctx *schema.ExecContext, token string) (schema.NodeResult, error) {
	folderID, _ := ctx.Params["folderId"].(string)
	if folderID == "" {
		folderID = "0"
	}

	// Use Box's events API to check for new uploads
	streamPosition, _ := ctx.State["streamPosition"].(string)
	endpoint := "/events?stream_type=changes&limit=10"
	if streamPosition != "" {
		endpoint += "&stream_position=" + streamPosition
	}

	raw, _, err := boxCall(token, "GET", endpoint, nil)
	if err != nil {
		return schema.NodeResult{}, fmt.Errorf("box trigger poll failed: %w", err)
	}

	var result map[string]any
	if json.Unmarshal(raw, &result) != nil {
		return schema.NodeResult{}, fmt.Errorf("box: failed to parse response")
	}

	// Update stream position for next poll
	if pos, ok := result["next_stream_position"].(string); ok {
		ctx.State["streamPosition"] = pos
	}

	var out []schema.Item
	if entries, ok := result["entries"].([]any); ok {
		for _, e := range entries {
			if m, ok := e.(map[string]any); ok {
				// Filter for ITEM_UPLOAD events in the target folder
				eventType, _ := m["event_type"].(string)
				if eventType == "ITEM_UPLOAD" || eventType == "ITEM_CREATE" {
					if source, ok := m["source"].(map[string]any); ok {
						if parent, ok := source["parent"].(map[string]any); ok {
							parentID, _ := parent["id"].(string)
							if folderID == "0" || parentID == folderID {
								out = append(out, schema.Item{JSON: source})
							}
						}
					}
				}
			}
		}
	}
	if out == nil {
		out = []schema.Item{}
	}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": out}}, nil
}
