package protocols

import (
	"testing"
)

func TestKafkaNode(t *testing.T) {
	def := Kafka()
	if def.Type != "protocols.kafka" {
		t.Fatalf("unexpected type: %s", def.Type)
	}
	if def.Execute == nil {
		t.Fatal("expected execute function to be set")
	}

	ops := collectOps(def)
	for _, want := range []string{"produce", "consume"} {
		if !ops[want] {
			t.Fatalf("expected operation %q", want)
		}
	}
}

func TestKafkaHasSASLParams(t *testing.T) {
	def := Kafka()
	hasSASL := false
	for _, p := range def.Params {
		if p.Name == "saslMechanism" {
			hasSASL = true
			if len(p.Options) < 3 {
				t.Fatalf("expected at least 3 SASL options, got %d", len(p.Options))
			}
			break
		}
	}
	if !hasSASL {
		t.Fatal("expected saslMechanism param")
	}
}

func TestKafkaHasGroupIdParam(t *testing.T) {
	def := Kafka()
	hasGroupID := false
	for _, p := range def.Params {
		if p.Name == "groupId" {
			hasGroupID = true
			break
		}
	}
	if !hasGroupID {
		t.Fatal("expected groupId param for consumer groups")
	}
}
