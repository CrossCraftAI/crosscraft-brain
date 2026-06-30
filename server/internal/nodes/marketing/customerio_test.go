package marketing

import (
	"testing"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

func TestCustomerIONode(t *testing.T) {
	def := CustomerIO().Build()
	if def.Type != "marketing.customerio" {
		t.Fatalf("unexpected type: %s", def.Type)
	}
	ops := collectOps(def)
	if len(ops) < 7 {
		t.Fatalf("expected at least 7 ops, got %d", len(ops))
	}
	for _, want := range []string{"customer:list", "customer:create", "customer:delete",
		"event:track", "campaign:list", "campaign:trigger",
		"message:list"} {
		if !ops[want] {
			t.Fatalf("expected operation %q", want)
		}
	}
}

func TestCustomerIOBaseURLParam(t *testing.T) {
	def := CustomerIO().Build()
	hasBaseURL := false
	for _, p := range def.Params {
		if p.Name == "baseUrl" {
			hasBaseURL = true
			break
		}
	}
	if !hasBaseURL {
		t.Fatal("expected baseUrl param for API endpoint switching")
	}
}

func collectOps(def schema.NodeDefinition) map[string]bool {
	ops := map[string]bool{}
	for _, p := range def.Params {
		if p.Name == "operation" {
			for _, opt := range p.Options {
				ops[opt.Value] = true
			}
		}
	}
	return ops
}
