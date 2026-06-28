package aws

import (
	"encoding/json"
	"fmt"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// DynamoDBNode provides DynamoDB item operations.
func DynamoDBNode() schema.NodeDefinition {
	return schema.NodeDefinition{
		Type: "aws.dynamodb", Label: "AWS DynamoDB", Group: "integration", Icon: "Database",
		Description: "Get, put, update, delete, and query DynamoDB items.",
		Inputs:  []schema.Port{{ID: "main"}},
		Outputs: []schema.Port{{ID: "main"}},
		Credentials: []string{"awsIam"},
		Params: []schema.ParamSchema{
			{Name: "credential", Label: "Credential", Type: "credential", Required: true, CredentialType: "awsIam"},
			{Name: "operation", Label: "Operation", Type: "select", Required: true, Options: []schema.ParamOption{
				{Label: "Get Item", Value: "get"},
				{Label: "Put Item", Value: "put"},
				{Label: "Update Item", Value: "update"},
				{Label: "Delete Item", Value: "delete"},
				{Label: "Query Items", Value: "query"},
				{Label: "Scan Table", Value: "scan"},
				{Label: "List Tables", Value: "listTables"},
			}},
			{Name: "tableName", Label: "Table name", Type: "string",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"get", "put", "update", "delete", "query", "scan"}}},
			{Name: "key", Label: "Key (JSON)", Type: "json",
				Description: `Primary key, e.g. {"id": {"S":"123"}} for string key or {"id":{"N":"1"}} for number.`,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"get", "delete"}}},
			{Name: "item", Label: "Item (JSON)", Type: "json",
				Description: `DynamoDB item, e.g. {"id":{"S":"123"},"name":{"S":"Alice"},"age":{"N":"30"}}`,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"put"}}},
			{Name: "updateKey", Label: "Key (JSON)", Type: "json",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"update"}}},
			{Name: "updateExpression", Label: "Update expression", Type: "string",
				Placeholder: "SET #n = :val",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"update"}}},
			{Name: "expressionAttributeNames", Label: "Expression attribute names (JSON)", Type: "json",
				Description: `{"#n": "name"}`,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"update"}}},
			{Name: "expressionAttributeValues", Label: "Expression attribute values (JSON)", Type: "json",
				Description: `{":val": {"S": "Updated Name"}}`,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"update"}}},
			{Name: "keyConditionExpression", Label: "Key condition expression", Type: "string",
				Placeholder: "id = :id",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"query"}}},
			{Name: "filterExpression", Label: "Filter expression", Type: "string",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"query", "scan"}}},
			{Name: "expressionValues", Label: "Expression values (JSON)", Type: "json",
				Description: `{":id":{"S":"123"}}`,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"query"}}},
			{Name: "limit", Label: "Max results", Type: "number", Default: float64(50),
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"query", "scan"}}},
		},
		Execute: func(ctx *schema.ExecContext) (schema.NodeResult, error) {
			signer, err := makeSigner(ctx, "dynamodb")
			if err != nil {
				return schema.NodeResult{}, err
			}
			op := asString(ctx.Params["operation"], "")
			region := signer.Region
			host := "dynamodb." + region + ".amazonaws.com"
			baseURL := "https://" + host

			switch op {

			case "listTables":
				target := "DynamoDB_20120810.ListTables"
				items, err := dynamoDo(signer, baseURL, target, map[string]any{})
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": items}}, err

			case "get":
				tableName := asString(ctx.Params["tableName"], "")
				key := ctx.RawParam("key")
				if tableName == "" || key == nil {
					return schema.NodeResult{}, fmt.Errorf("dynamodb get: tableName and key are required")
				}
				target := "DynamoDB_20120810.GetItem"
				body := map[string]any{"TableName": tableName, "Key": key}
				items, err := dynamoDo(signer, baseURL, target, body)
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": items}}, err

			case "put":
				tableName := asString(ctx.Params["tableName"], "")
				item := ctx.RawParam("item")
				if tableName == "" || item == nil {
					return schema.NodeResult{}, fmt.Errorf("dynamodb put: tableName and item are required")
				}
				target := "DynamoDB_20120810.PutItem"
				body := map[string]any{"TableName": tableName, "Item": item}
				items, err := dynamoDo(signer, baseURL, target, body)
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": items}}, err

			case "update":
				tableName := asString(ctx.Params["tableName"], "")
				updKey := ctx.RawParam("updateKey")
				updExpr := asString(ctx.Params["updateExpression"], "")
				if tableName == "" || updKey == nil || updExpr == "" {
					return schema.NodeResult{}, fmt.Errorf("dynamodb update: tableName, updateKey, and updateExpression are required")
				}
				target := "DynamoDB_20120810.UpdateItem"
				body := map[string]any{
					"TableName":        tableName,
					"Key":              updKey,
					"UpdateExpression": updExpr,
				}
				if attrNames := ctx.RawParam("expressionAttributeNames"); attrNames != nil {
					body["ExpressionAttributeNames"] = attrNames
				}
				if attrVals := ctx.RawParam("expressionAttributeValues"); attrVals != nil {
					body["ExpressionAttributeValues"] = attrVals
				}
				items, err := dynamoDo(signer, baseURL, target, body)
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": items}}, err

			case "delete":
				tableName := asString(ctx.Params["tableName"], "")
				key := ctx.RawParam("key")
				if tableName == "" || key == nil {
					return schema.NodeResult{}, fmt.Errorf("dynamodb delete: tableName and key are required")
				}
				target := "DynamoDB_20120810.DeleteItem"
				body := map[string]any{"TableName": tableName, "Key": key}
				items, err := dynamoDo(signer, baseURL, target, body)
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": items}}, err

			case "query":
				tableName := asString(ctx.Params["tableName"], "")
				keyCond := asString(ctx.Params["keyConditionExpression"], "")
				if tableName == "" || keyCond == "" {
					return schema.NodeResult{}, fmt.Errorf("dynamodb query: tableName and keyConditionExpression are required")
				}
				target := "DynamoDB_20120810.Query"
				body := map[string]any{
					"TableName":                tableName,
					"KeyConditionExpression":   keyCond,
					"Limit":                    asInt(ctx.Params["limit"], 50),
				}
				if exprVals := ctx.RawParam("expressionValues"); exprVals != nil {
					body["ExpressionAttributeValues"] = exprVals
				}
				if filterExpr := asString(ctx.Params["filterExpression"], ""); filterExpr != "" {
					body["FilterExpression"] = filterExpr
				}
				items, err := dynamoDo(signer, baseURL, target, body)
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": items}}, err

			case "scan":
				tableName := asString(ctx.Params["tableName"], "")
				if tableName == "" {
					return schema.NodeResult{}, fmt.Errorf("dynamodb scan: tableName is required")
				}
				target := "DynamoDB_20120810.Scan"
				body := map[string]any{
					"TableName": tableName,
					"Limit":     asInt(ctx.Params["limit"], 50),
				}
				if filterExpr := asString(ctx.Params["filterExpression"], ""); filterExpr != "" {
					body["FilterExpression"] = filterExpr
				}
				items, err := dynamoDo(signer, baseURL, target, body)
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": items}}, err

			default:
				return schema.NodeResult{}, fmt.Errorf("dynamodb: unknown operation %q", op)
			}
		},
	}
}

func dynamoDo(signer *Signer, baseURL, target string, body map[string]any) ([]schema.Item, error) {
	body["__typename"] = target
	bodyBytes, _ := json.Marshal(body)

	items, err := awsDo(signer, "POST", baseURL, bodyBytes)
	if err != nil {
		return items, err
	}
	// Unwrap DynamoDB typed attributes to plain JSON
	for i := range items {
		items[i].JSON = dynamoUnwrapAny(items[i].JSON)
	}
	return items, nil
}

// dynamoUnwrapAny recursively converts DynamoDB typed attributes
// ({"S":"value"}, {"N":"123"}, etc.) to plain Go values.
func dynamoUnwrapAny(v any) map[string]any {
	switch t := v.(type) {
	case map[string]any:
		out := map[string]any{}
		for k, val := range t {
			if k == "Item" || k == "Items" || k == "Attributes" {
				if m, ok := val.(map[string]any); ok {
					out[k] = dynamoUnwrapMap(m)
					continue
				}
				if arr, ok := val.([]any); ok {
					items := make([]map[string]any, 0, len(arr))
					for _, e := range arr {
						if em, ok := e.(map[string]any); ok {
							items = append(items, dynamoUnwrapMap(em))
						}
					}
					out[k] = items
					continue
				}
			}
			out[k] = dynamoUnwrapMap(map[string]any{k: val})[k]
		}
		return out
	default:
		return map[string]any{"value": v}
	}
}

func dynamoUnwrapMap(m map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range m {
		if vm, ok := v.(map[string]any); ok {
			if s, ok := vm["S"].(string); ok {
				out[k] = s
			} else if n, ok := vm["N"].(string); ok {
				// Try to parse as number
				var f float64
				if _, err := fmt.Sscanf(n, "%f", &f); err == nil {
					if n == fmt.Sprint(int64(f)) && !contains(n, ".") {
						out[k] = int64(f)
					} else {
						out[k] = f
					}
				} else {
					out[k] = n
				}
			} else if b, ok := vm["BOOL"].(bool); ok {
				out[k] = b
			} else if vm["NULL"] != nil {
				out[k] = nil
			} else if l, ok := vm["L"].([]any); ok {
				unwrapped := make([]any, 0, len(l))
				for _, item := range l {
					if im, ok := item.(map[string]any); ok {
						unwrapped = append(unwrapped, dynamoUnwrapMap(im)[k])
					} else {
						unwrapped = append(unwrapped, item)
					}
				}
				out[k] = unwrapped
			} else if subMap, ok := vm["M"].(map[string]any); ok {
				out[k] = dynamoUnwrapMap(subMap)
			} else {
				out[k] = v
			}
		} else {
			out[k] = v
		}
	}
	return out
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchStr(s, substr)
}

func searchStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
