package crm

import (
	"testing"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

func TestSalesforceNode(t *testing.T) {
	def := Salesforce("https://test.salesforce.com").Build()
	if def.Type != "crm.salesforce" {
		t.Fatalf("unexpected type: %s", def.Type)
	}
	ops := collectOps(def)
	if len(ops) < 10 {
		t.Fatalf("expected at least 10 ops, got %d", len(ops))
	}
	for _, want := range []string{"account:list", "account:create", "contact:list", "lead:list", "opportunity:list", "query:execute", "sobject:describe"} {
		if !ops[want] {
			t.Fatalf("expected operation %q", want)
		}
	}
}

func TestCRMNodeCount(t *testing.T) {
	nodes := Nodes()
	if len(nodes) != 5 {
		t.Fatalf("expected 5 CRM nodes, got %d", len(nodes))
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
