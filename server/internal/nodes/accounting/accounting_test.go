package accounting

import (
	"testing"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

func TestQuickBooksNode(t *testing.T) {
	def := QuickBooks("https://quickbooks.api.intuit.com").Build()
	ops := collectOps(def)
	if len(ops) < 8 {
		t.Fatalf("expected at least 8 ops, got %d", len(ops))
	}
	for _, want := range []string{"invoice:list", "invoice:create", "customer:list", "expense:list", "report:profitLoss"} {
		if !ops[want] {
			t.Fatalf("expected operation %q", want)
		}
	}
}

func TestXeroNode(t *testing.T) {
	def := Xero("https://api.xero.com").Build()
	ops := collectOps(def)
	if len(ops) < 8 {
		t.Fatalf("expected at least 8 ops, got %d", len(ops))
	}
	for _, want := range []string{"invoice:list", "invoice:create", "contact:list", "bankTransaction:list", "report:profitLoss", "account:list"} {
		if !ops[want] {
			t.Fatalf("expected operation %q", want)
		}
	}
}

func TestAccountingNodesRegistration(t *testing.T) {
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
