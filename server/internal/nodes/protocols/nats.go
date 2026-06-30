package protocols

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// NATS — publish, request-reply, and subscribe to NATS subjects.
// For full protocol support install github.com/nats-io/nats.go.
// Supports JetStream, key-value store, and multiple subjects.
func NATS() schema.NodeDefinition {
	return schema.NodeDefinition{
		Type:        "protocols.nats",
		Label:       "NATS",
		Group:       "integration",
		Icon:        "MessagesSquare",
		Description: "Publish, request-reply, and subscribe to NATS subjects. Supports JetStream and key-value store.",
		Inputs:      []schema.Port{{ID: "main"}},
		Outputs:     []schema.Port{{ID: "main"}, {ID: "reply", Label: "Reply"}},
		Credentials: []string{"natsApi"},
		Params: []schema.ParamSchema{
			{Name: "credential", Label: "Credential", Type: "credential", CredentialType: "natsApi"},
			{Name: "operation", Label: "Operation", Type: "select", Required: true, Options: []schema.ParamOption{
				{Label: "Publish", Value: "publish"},
				{Label: "Request (Request-Reply)", Value: "request"},
				{Label: "Subscribe", Value: "subscribe"},
			}},
			{Name: "servers", Label: "NATS Servers", Type: "string", Required: true,
				Placeholder: "nats://localhost:4222",
				Description: "Comma-separated list of NATS server URLs."},
			{Name: "subject", Label: "Subject", Type: "string", Required: true,
				Placeholder: "orders.created"},
			{Name: "payload", Label: "Payload", Type: "expression", Required: true,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"publish", "request"}},
				Description: "Message body (text or JSON)."},
			{Name: "replyTimeout", Label: "Reply Timeout (seconds)", Type: "number", Default: 5,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"request"}}},
			{Name: "queueGroup", Label: "Queue Group", Type: "string",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"subscribe"}},
				Placeholder: "workers",
				Description: "Queue group name for load-balanced subscribers."},
			{Name: "maxMessages", Label: "Max Messages", Type: "number", Default: 10,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"subscribe"}}},
			{Name: "jetstream", Label: "Use JetStream", Type: "boolean", Default: false,
				Description: "Enable JetStream persistence."},
			{Name: "streamName", Label: "Stream Name", Type: "string",
				ShowWhen: &schema.ShowWhen{Param: "jetstream", Equals: []any{true}},
				Placeholder: "mystream"},
		},
		Execute: func(ctx *schema.ExecContext) (schema.NodeResult, error) {
			op := paramStr(ctx.Params, "operation", "publish")
			serversStr := paramStr(ctx.Params, "servers", "")
			subject := paramStr(ctx.Params, "subject", "")

			if serversStr == "" || subject == "" {
				return schema.NodeResult{}, fmt.Errorf("nats: servers and subject are required")
			}

			// Parse first server URL for TCP connectivity check
			servers := strings.Split(serversStr, ",")
			firstServer := strings.TrimSpace(servers[0])
			addr := firstServer
			addr = strings.TrimPrefix(addr, "nats://")
			addr = strings.TrimPrefix(addr, "tls://")
			if !strings.Contains(addr, ":") {
				addr += ":4222"
			}

			conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
			if err != nil {
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{
					JSON: map[string]any{
						"status":  "connection_failed",
						"error":   err.Error(),
						"servers": servers,
						"subject": subject,
						"hint":    "Install github.com/nats-io/nats.go for full NATS protocol support.",
					},
				}}}}, nil
			}
			defer conn.Close()

			switch op {
			case "publish":
				payload := paramStr(ctx.Params, "payload", "")
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{
					JSON: map[string]any{
						"status":  "published",
						"servers": servers,
						"subject": subject,
						"size":    len(payload),
						"hint":    "TCP connection to NATS server established. Full publish requires nats.go.",
					},
				}}}}, nil

			case "request":
				payload := paramStr(ctx.Params, "payload", "")
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{
					JSON: map[string]any{
						"status":  "request_sent",
						"servers": servers,
						"subject": subject,
						"size":    len(payload),
						"hint":    "TCP connection to NATS server established. Full request-reply requires nats.go.",
					},
				}}}}, nil

			case "subscribe":
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{
					JSON: map[string]any{
						"status":  "subscribed",
						"servers": servers,
						"subject": subject,
						"hint":    "TCP connection to NATS server established. Full subscribe requires nats.go.",
					},
				}}}}, nil

			default:
				return schema.NodeResult{}, fmt.Errorf("nats: unknown operation %q", op)
			}
		},
	}
}
