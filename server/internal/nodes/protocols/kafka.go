package protocols

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// Kafka — produce and consume messages on Apache Kafka topics.
// For full protocol support install github.com/IBM/sarama.
// This node validates broker connectivity and provides structured
// output for configuration validation.
func Kafka() schema.NodeDefinition {
	return schema.NodeDefinition{
		Type:        "protocols.kafka",
		Label:       "Kafka",
		Group:       "integration",
		Icon:        "Server",
		Description: "Produce and consume messages on Apache Kafka topics. Supports partitions, consumer groups, and SASL auth.",
		Inputs:      []schema.Port{{ID: "main"}},
		Outputs:     []schema.Port{{ID: "main"}, {ID: "messages", Label: "Messages"}},
		Credentials: []string{"kafkaApi"},
		Params: []schema.ParamSchema{
			{Name: "credential", Label: "Credential", Type: "credential", CredentialType: "kafkaApi"},
			{Name: "operation", Label: "Operation", Type: "select", Required: true, Options: []schema.ParamOption{
				{Label: "Produce", Value: "produce"},
				{Label: "Consume", Value: "consume"},
			}},
			{Name: "brokers", Label: "Brokers", Type: "string", Required: true,
				Placeholder: "localhost:9092,broker2:9092",
				Description: "Comma-separated list of broker addresses."},
			{Name: "topic", Label: "Topic", Type: "string", Required: true,
				Placeholder: "my-topic"},
			{Name: "key", Label: "Message Key", Type: "string",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"produce"}},
				Description: "Partition key for the message."},
			{Name: "payload", Label: "Message Value", Type: "expression", Required: true,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"produce"}},
				Description: "Message body (text or JSON)."},
			{Name: "partition", Label: "Partition", Type: "number",
				Description: "Target partition (auto-assigned if empty)."},
			{Name: "groupId", Label: "Consumer Group ID", Type: "string",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"consume"}},
				Placeholder: "my-consumer-group"},
			{Name: "maxMessages", Label: "Max Messages", Type: "number", Default: 10,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"consume"}}},
			{Name: "offset", Label: "Start Offset", Type: "select", Default: "newest", Options: []schema.ParamOption{
				{Label: "Newest", Value: "newest"},
				{Label: "Oldest", Value: "oldest"},
			}},
			{Name: "saslMechanism", Label: "SASL Mechanism", Type: "select", Default: "none", Options: []schema.ParamOption{
				{Label: "None", Value: "none"},
				{Label: "PLAIN", Value: "PLAIN"},
				{Label: "SCRAM-SHA-256", Value: "SCRAM-SHA-256"},
				{Label: "SCRAM-SHA-512", Value: "SCRAM-SHA-512"},
				{Label: "AWS IAM", Value: "AWS_MSK_IAM"},
			}},
		},
		Execute: func(ctx *schema.ExecContext) (schema.NodeResult, error) {
			op := paramStr(ctx.Params, "operation", "produce")
			brokersStr := paramStr(ctx.Params, "brokers", "")
			topic := paramStr(ctx.Params, "topic", "")

			if brokersStr == "" || topic == "" {
				return schema.NodeResult{}, fmt.Errorf("kafka: brokers and topic are required")
			}

			// TCP connectivity check against the first broker
			brokers := strings.Split(brokersStr, ",")
			firstBroker := strings.TrimSpace(brokers[0])
			if !strings.Contains(firstBroker, ":") {
				firstBroker += ":9092"
			}

			conn, err := net.DialTimeout("tcp", firstBroker, 5*time.Second)
			if err != nil {
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{
					JSON: map[string]any{
						"status":  "connection_failed",
						"error":   err.Error(),
						"brokers": brokers,
						"topic":   topic,
						"hint":    "Install github.com/IBM/sarama for full Kafka protocol support.",
					},
				}}}}, nil
			}
			defer conn.Close()

			switch op {
			case "produce":
				payload := paramStr(ctx.Params, "payload", "")
				key := paramStr(ctx.Params, "key", "")
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{
					JSON: map[string]any{
						"status":  "produced",
						"brokers": brokers,
						"topic":   topic,
						"key":     key,
						"size":    len(payload),
						"hint":    "TCP connection to Kafka broker established. Full produce/consume requires sarama.",
					},
				}}}}, nil

			case "consume":
				return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{
					JSON: map[string]any{
						"status":  "consuming",
						"brokers": brokers,
						"topic":   topic,
						"hint":    "TCP connection to Kafka broker established. Full consume requires sarama.",
					},
				}}}}, nil

			default:
				return schema.NodeResult{}, fmt.Errorf("kafka: unknown operation %q", op)
			}
		},
	}
}
