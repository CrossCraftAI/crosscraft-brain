package aws

import (
	"fmt"
	"net/url"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// SQSNode provides Amazon SQS message queue operations.
func SQSNode() schema.NodeDefinition {
	return schema.NodeDefinition{
		Type: "aws.sqs", Label: "AWS SQS", Group: "integration", Icon: "MessageSquare",
		Description: "Send, receive, and manage SQS messages.",
		Inputs:  []schema.Port{{ID: "main"}},
		Outputs: []schema.Port{{ID: "main"}},
		Credentials: []string{"awsIam"},
		Params: []schema.ParamSchema{
			{Name: "credential", Label: "Credential", Type: "credential", Required: true, CredentialType: "awsIam"},
			{Name: "operation", Label: "Operation", Type: "select", Required: true, Options: []schema.ParamOption{
				{Label: "List Queues", Value: "listQueues"},
				{Label: "Send Message", Value: "send"},
				{Label: "Receive Messages", Value: "receive"},
				{Label: "Delete Message", Value: "delete"},
			}},
			{Name: "queueURL", Label: "Queue URL", Type: "string",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"send", "receive", "delete"}}},
			{Name: "messageBody", Label: "Message body", Type: "code",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"send"}}},
			{Name: "maxMessages", Label: "Max messages", Type: "number", Default: float64(10),
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"receive"}}},
			{Name: "visibilityTimeout", Label: "Visibility timeout (seconds)", Type: "number", Default: float64(30),
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"receive"}}},
			{Name: "waitTimeSeconds", Label: "Wait time (seconds)", Type: "number", Default: float64(0),
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"receive"}}},
			{Name: "receiptHandle", Label: "Receipt handle", Type: "string",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"delete"}}},
		},
		Execute: func(ctx *schema.ExecContext) (schema.NodeResult, error) {
			signer, err := makeSigner(ctx, "sqs")
			if err != nil {
				return schema.NodeResult{}, err
			}
			op := asString(ctx.Params["operation"], "")
			region := signer.Region

			switch op {

			case "listQueues":
				host := "sqs." + region + ".amazonaws.com"
				params := url.Values{"Action": {"ListQueues"}, "Version": {"2012-11-05"}}
				items, err := sqsQuery(signer, "https://"+host, params)
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": items}}, err

			case "send":
				queueURL := asString(ctx.Params["queueURL"], "")
				msgBody := asString(ctx.Params["messageBody"], "")
				if queueURL == "" || msgBody == "" {
					return schema.NodeResult{}, fmt.Errorf("sqs send: queueURL and messageBody are required")
				}
				params := url.Values{
					"Action":      {"SendMessage"},
					"Version":     {"2012-11-05"},
					"MessageBody": {msgBody},
				}
				items, err := sqsQuery(signer, queueURL, params)
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": items}}, err

			case "receive":
				queueURL := asString(ctx.Params["queueURL"], "")
				if queueURL == "" {
					return schema.NodeResult{}, fmt.Errorf("sqs receive: queueURL is required")
				}
				maxMsg := asInt(ctx.Params["maxMessages"], 10)
				visTimeout := asInt(ctx.Params["visibilityTimeout"], 30)
				waitTime := asInt(ctx.Params["waitTimeSeconds"], 0)
				params := url.Values{
					"Action":              {"ReceiveMessage"},
					"Version":             {"2012-11-05"},
					"MaxNumberOfMessages": {fmt.Sprint(maxMsg)},
					"VisibilityTimeout":   {fmt.Sprint(visTimeout)},
					"WaitTimeSeconds":     {fmt.Sprint(waitTime)},
				}
				items, err := sqsQuery(signer, queueURL, params)
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": items}}, err

			case "delete":
				queueURL := asString(ctx.Params["queueURL"], "")
				handle := asString(ctx.Params["receiptHandle"], "")
				if queueURL == "" || handle == "" {
					return schema.NodeResult{}, fmt.Errorf("sqs delete: queueURL and receiptHandle are required")
				}
				params := url.Values{
					"Action":        {"DeleteMessage"},
					"Version":       {"2012-11-05"},
					"ReceiptHandle": {handle},
				}
				items, err := sqsQuery(signer, queueURL, params)
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": items}}, err

			default:
				return schema.NodeResult{}, fmt.Errorf("sqs: unknown operation %q", op)
			}
		},
	}
}

func sqsQuery(signer *Signer, baseURL string, params url.Values) ([]schema.Item, error) {
	return sesQuery(signer, baseURL, params) // Reuse SES query helper (identical XML request pattern)
}
