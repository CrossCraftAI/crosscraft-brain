package protocols

import (
	"testing"
)

func TestMQTTNode(t *testing.T) {
	def := MQTT()
	if def.Type != "protocols.mqtt" {
		t.Fatalf("unexpected type: %s", def.Type)
	}
	if def.Execute == nil {
		t.Fatal("expected execute function to be set")
	}

	ops := collectOps(def)
	for _, want := range []string{"publish", "subscribe"} {
		if !ops[want] {
			t.Fatalf("expected operation %q", want)
		}
	}
}

func TestMQTTHasQoSParam(t *testing.T) {
	def := MQTT()
	hasQoS := false
	hasRetain := false
	for _, p := range def.Params {
		if p.Name == "qos" {
			hasQoS = true
		}
		if p.Name == "retain" {
			hasRetain = true
		}
	}
	if !hasQoS {
		t.Fatal("expected qos param")
	}
	if !hasRetain {
		t.Fatal("expected retain param")
	}
}

func TestMQTTHasWillParams(t *testing.T) {
	def := MQTT()
	hasWillTopic := false
	hasWillPayload := false
	for _, p := range def.Params {
		if p.Name == "willTopic" {
			hasWillTopic = true
		}
		if p.Name == "willPayload" {
			hasWillPayload = true
		}
	}
	if !hasWillTopic {
		t.Fatal("expected willTopic param for Last Will and Testament")
	}
	if !hasWillPayload {
		t.Fatal("expected willPayload param")
	}
}
