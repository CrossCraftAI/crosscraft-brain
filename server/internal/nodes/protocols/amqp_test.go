package protocols

import (
	"testing"
)

func TestAMQPNode(t *testing.T) {
	def := AMQP()
	if def.Type != "protocols.amqp" {
		t.Fatalf("unexpected type: %s", def.Type)
	}
	if def.Execute == nil {
		t.Fatal("expected execute function to be set")
	}

	ops := collectOps(def)
	for _, want := range []string{"publish", "consume"} {
		if !ops[want] {
			t.Fatalf("expected operation %q", want)
		}
	}
}

func TestAMQPHasExchangeTypes(t *testing.T) {
	def := AMQP()
	hasExchangeType := false
	for _, p := range def.Params {
		if p.Name == "exchangeType" {
			hasExchangeType = true
			if len(p.Options) != 4 {
				t.Fatalf("expected 4 exchange types, got %d", len(p.Options))
			}
			break
		}
	}
	if !hasExchangeType {
		t.Fatal("expected exchangeType param")
	}
}

func TestAMQPHasDeadLetterParam(t *testing.T) {
	def := AMQP()
	hasDLX := false
	for _, p := range def.Params {
		if p.Name == "deadLetterExchange" {
			hasDLX = true
			break
		}
	}
	if !hasDLX {
		t.Fatal("expected deadLetterExchange param")
	}
}
