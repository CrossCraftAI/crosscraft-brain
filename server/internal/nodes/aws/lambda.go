package aws

import (
	"encoding/json"
	"fmt"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// LambdaNode provides AWS Lambda function invocation.
func LambdaNode() schema.NodeDefinition {
	return schema.NodeDefinition{
		Type: "aws.lambda", Label: "AWS Lambda", Group: "integration", Icon: "Code",
		Description: "Invoke AWS Lambda functions with JSON payloads.",
		Inputs:  []schema.Port{{ID: "main"}},
		Outputs: []schema.Port{{ID: "main"}},
		Credentials: []string{"awsIam"},
		Params: []schema.ParamSchema{
			{Name: "credential", Label: "Credential", Type: "credential", Required: true, CredentialType: "awsIam"},
			{Name: "operation", Label: "Operation", Type: "select", Required: true, Options: []schema.ParamOption{
				{Label: "Invoke Function", Value: "invoke"},
				{Label: "List Functions", Value: "list"},
			}},
			{Name: "functionName", Label: "Function name or ARN", Type: "string",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"invoke"}}},
			{Name: "payload", Label: "Payload (JSON)", Type: "json",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"invoke"}}},
			{Name: "qualifier", Label: "Qualifier (alias/version)", Type: "string",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"invoke"}}},
			{Name: "invocationType", Label: "Invocation type", Type: "select", Default: "RequestResponse",
				Options: []schema.ParamOption{
					{Label: "RequestResponse", Value: "RequestResponse"},
					{Label: "Event (async)", Value: "Event"},
					{Label: "DryRun", Value: "DryRun"},
				},
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"invoke"}}},
		},
		Execute: func(ctx *schema.ExecContext) (schema.NodeResult, error) {
			signer, err := makeSigner(ctx, "lambda")
			if err != nil {
				return schema.NodeResult{}, err
			}
			op := asString(ctx.Params["operation"], "")
			region := signer.Region
			host := "lambda." + region + ".amazonaws.com"

			switch op {

			case "invoke":
				fnName := asString(ctx.Params["functionName"], "")
				if fnName == "" {
					return schema.NodeResult{}, fmt.Errorf("lambda invoke: functionName is required")
				}
				payload := ctx.RawParam("payload")
				payloadBytes, _ := json.Marshal(payload)
				if payloadBytes == nil {
					payloadBytes = []byte("{}")
				}
				path := "/2015-03-31/functions/" + fnName + "/invocations"
				qualifier := asString(ctx.Params["qualifier"], "")
				if qualifier != "" {
					path += "?Qualifier=" + qualifier
				}
				invType := asString(ctx.Params["invocationType"], "RequestResponse")

				items, err := awsDo(signer, "POST", "https://"+host+path, payloadBytes)
				if err != nil {
					return schema.NodeResult{}, err
				}
				// Add invocation metadata
				for i := range items {
					items[i].JSON["functionName"] = fnName
					items[i].JSON["invocationType"] = invType
				}
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": items}}, err

			case "list":
				items, err := awsDo(signer, "GET", "https://"+host+"/2015-03-31/functions/", nil)
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": items}}, err

			default:
				return schema.NodeResult{}, fmt.Errorf("lambda: unknown operation %q", op)
			}
		},
	}
}
