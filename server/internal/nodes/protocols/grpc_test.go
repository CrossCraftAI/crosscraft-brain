package protocols

import (
	"testing"
)

func TestGRPCNode(t *testing.T) {
	def := GRPC()
	if def.Type != "protocols.grpc" {
		t.Fatalf("unexpected type: %s", def.Type)
	}
	if def.Execute == nil {
		t.Fatal("expected execute function to be set")
	}
	if len(def.Outputs) < 2 {
		t.Fatal("expected main + stream outputs")
	}

	ops := collectOps(def)
	for _, want := range []string{"unary", "serverStream", "clientStream", "bidirectionalStream"} {
		if !ops[want] {
			t.Fatalf("expected operation %q", want)
		}
	}
}

func TestGRPCHasTLSParam(t *testing.T) {
	def := GRPC()
	hasTLS := false
	for _, p := range def.Params {
		if p.Name == "tls" {
			hasTLS = true
			break
		}
	}
	if !hasTLS {
		t.Fatal("expected tls param")
	}
}
