package payments

import (
	"testing"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

func TestPayPalNode(t *testing.T) {
	def := PayPal("https://api-m.paypal.com").Build()
	ops := collectOps(def)
	if len(ops) < 8 {
		t.Fatalf("expected at least 8 ops, got %d", len(ops))
	}
	for _, want := range []string{"order:create", "order:capture", "payment:get", "refund:create", "webhook:list", "invoice:create"} {
		if !ops[want] {
			t.Fatalf("expected operation %q", want)
		}
	}
}

func TestSquareNode(t *testing.T) {
	def := Square("https://connect.squareup.com").Build()
	ops := collectOps(def)
	if len(ops) < 8 {
		t.Fatalf("expected at least 8 ops, got %d", len(ops))
	}
	for _, want := range []string{"payment:create", "order:create", "customer:list", "refund:create"} {
		if !ops[want] {
			t.Fatalf("expected operation %q", want)
		}
	}
}

func TestPaymentsNodesRegistration(t *testing.T) {
	nodes := Nodes()
	if len(nodes) != 3 {
		t.Fatalf("expected 3 nodes (Stripe, PayPal, Square), got %d", len(nodes))
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
