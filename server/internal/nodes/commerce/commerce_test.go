package commerce

import (
	"testing"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

func TestShopifyNode(t *testing.T) {
	def := Shopify("https://test.myshopify.com").Build()
	ops := collectOps(def)
	if len(ops) < 10 {
		t.Fatalf("expected at least 10 ops, got %d", len(ops))
	}
	for _, want := range []string{"product:list", "product:create", "order:list", "customer:list", "draftOrder:create"} {
		if !ops[want] {
			t.Fatalf("expected operation %q", want)
		}
	}
}

func TestWooCommerceNode(t *testing.T) {
	def := WooCommerce("https://test.com").Build()
	ops := collectOps(def)
	if len(ops) < 10 {
		t.Fatalf("expected at least 10 ops, got %d", len(ops))
	}
	for _, want := range []string{"product:list", "product:create", "order:list", "customer:list", "coupon:list"} {
		if !ops[want] {
			t.Fatalf("expected operation %q", want)
		}
	}
}

func TestCommerceNodesRegistration(t *testing.T) {
	nodes := Nodes()
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
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
