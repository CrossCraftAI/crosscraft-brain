// Package protocols provides generic protocol integration nodes: GraphQL, gRPC,
// SOAP, MQTT, AMQP/RabbitMQ, Kafka, NATS, and WebSocket. Nodes mix declarative
// REST (for HTTP-based protocols) and native implementations where protocol-level
// client libraries are required.
package protocols

import (
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

// Nodes returns the full generic-protocol node pack.
func Nodes() []schema.NodeDefinition {
	return []schema.NodeDefinition{
		GraphQL(),
		GRPC(),
		SOAP(),
		MQTT(),
		AMQP(),
		Kafka(),
		NATS(),
		WebSocket(),
	}
}
