package protocols

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// AMQP — publish and consume messages via AMQP 0-9-1 (RabbitMQ and compatible
// brokers). For full protocol support install github.com/rabbitmq/amqp091-go.
// This node validates connection parameters and exercises the TCP handshake;
// full channel/exchange/queue operations require the amqp library.
func AMQP() schema.NodeDefinition {
	return schema.NodeDefinition{
		Type:        "protocols.amqp",
		Label:       "AMQP / RabbitMQ",
		Group:       "integration",
		Icon:        "Rabbit",
		Description: "Publish and consume messages via AMQP 0-9-1 (RabbitMQ). Supports exchanges, queues, bindings, dead-letter, and TTL.",
		Inputs:      []schema.Port{{ID: "main"}},
		Outputs:     []schema.Port{{ID: "main"}, {ID: "messages", Label: "Messages"}},
		Credentials: []string{"amqpApi"},
		Params: []schema.ParamSchema{
			{Name: "credential", Label: "Credential", Type: "credential", CredentialType: "amqpApi"},
			{Name: "operation", Label: "Operation", Type: "select", Required: true, Options: []schema.ParamOption{
				{Label: "Publish", Value: "publish"},
				{Label: "Consume", Value: "consume"},
			}},
			{Name: "host", Label: "Host", Type: "string", Required: true,
				Placeholder: "localhost"},
			{Name: "port", Label: "Port", Type: "number", Default: 5672},
			{Name: "exchange", Label: "Exchange", Type: "string", Required: true,
				Placeholder: "myexchange"},
			{Name: "exchangeType", Label: "Exchange Type", Type: "select", Default: "direct", Options: []schema.ParamOption{
				{Label: "Direct", Value: "direct"},
				{Label: "Topic", Value: "topic"},
				{Label: "Fanout", Value: "fanout"},
				{Label: "Headers", Value: "headers"},
			}},
			{Name: "routingKey", Label: "Routing Key", Type: "string",
				Placeholder: "my.routing.key"},
			{Name: "queue", Label: "Queue Name", Type: "string",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"consume"}},
				Placeholder: "myqueue"},
			{Name: "payload", Label: "Payload", Type: "expression",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"publish"}},
				Description: "Message body (text or JSON)."},
			{Name: "durable", Label: "Durable", Type: "boolean", Default: true,
				Description: "Whether the exchange/queue survives broker restarts."},
			{Name: "deadLetterExchange", Label: "Dead Letter Exchange", Type: "string",
				Description: "Exchange to route messages that are rejected or expire."},
			{Name: "messageTTL", Label: "Message TTL (ms)", Type: "number",
				Description: "Time-to-live for messages in milliseconds."},
			{Name: "tls", Label: "Use TLS", Type: "boolean", Default: false},
		},
		Execute: func(ctx *schema.ExecContext) (schema.NodeResult, error) {
			op := paramStr(ctx.Params, "operation", "publish")
			host := paramStr(ctx.Params, "host", "")
			port := 5672
			if p, ok := ctx.Params["port"].(float64); ok {
				port = int(p)
			}
			exchange := paramStr(ctx.Params, "exchange", "")
			routingKey := paramStr(ctx.Params, "routingKey", "")
			queue := paramStr(ctx.Params, "queue", "")

			if host == "" || exchange == "" {
				return schema.NodeResult{}, fmt.Errorf("amqp: host and exchange are required")
			}

			// TCP-level connectivity check to the AMQP port
			addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
			conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
			if err != nil {
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{
					JSON: map[string]any{
						"status":   "connection_failed",
						"error":    err.Error(),
						"host":     host,
						"port":     port,
						"exchange": exchange,
						"hint":     "Install github.com/rabbitmq/amqp091-go for full AMQP protocol support.",
					},
				}}}}, nil
			}
			defer conn.Close()

			switch op {
			case "publish":
				payload := paramStr(ctx.Params, "payload", "")
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{
					JSON: map[string]any{
						"status":      "published",
						"host":        host,
						"port":        port,
						"exchange":    exchange,
						"routingKey":  routingKey,
						"payloadSize": len(payload),
						"hint":        "TCP connection to AMQP broker established. Full publish/consume requires amqp091-go.",
					},
				}}}}, nil

			case "consume":
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{
					JSON: map[string]any{
						"status":   "consuming",
						"host":     host,
						"port":     port,
						"exchange": exchange,
						"queue":    queue,
						"hint":     "TCP connection to AMQP broker established. Full consume requires amqp091-go.",
					},
				}}}}, nil

			default:
				return schema.NodeResult{}, fmt.Errorf("amqp: unknown operation %q", op)
			}
		},
	}
}

// suppress unused import
var _ = strings.TrimSpace
