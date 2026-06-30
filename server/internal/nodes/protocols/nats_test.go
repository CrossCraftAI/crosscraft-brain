package protocols

import (
	"testing"
)

func TestNATSNode(t *testing.T) {
	def := NATS()
	if def.Type != "protocols.nats" {
		t.Fatalf("unexpected type: %s", def.Type)
	}
	if def.Execute == nil {
		t.Fatal("expected execute function to be set")
	}
	if len(def.Outputs) != 2 {
		t.Fatalf("expected 2 outputs (main + reply), got %d", len(def.Outputs))
	}

	ops := collectOps(def)
	for _, want := range []string{"publish", "request", "subscribe"} {
		if !ops[want] {
			t.Fatalf("expected operation %q", want)
		}
	}
}

func TestNATSHasJetStreamParam(t *testing.T) {
	def := NATS()
	hasJS := false
	hasStream := false
	for _, p := range def.Params {
		if p.Name == "jetstream" {
			hasJS = true
		}
		if p.Name == "streamName" {
			hasStream = true
		}
	}
	if !hasJS {
		t.Fatal("expected jetstream param")
	}
	if !hasStream {
		t.Fatal("expected streamName param")
	}
}

func TestNATSHasQueueGroupParam(t *testing.T) {
	def := NATS()
	hasQG := false
	for _, p := range def.Params {
		if p.Name == "queueGroup" {
			hasQG = true
			break
		}
	}
	if !hasQG {
		t.Fatal("expected queueGroup param for load-balanced subscribers")
	}
}
