package crm

import (
	"testing"
)

func TestZohoCRMNode(t *testing.T) {
	def := ZohoCRM().Build()
	if def.Type != "crm.zoho" {
		t.Fatalf("unexpected type: %s", def.Type)
	}
	ops := collectOps(def)
	if len(ops) < 10 {
		t.Fatalf("expected at least 10 ops, got %d", len(ops))
	}
	for _, want := range []string{"lead:list", "lead:create", "lead:update", "lead:delete",
		"contact:list", "contact:create", "contact:update",
		"account:list", "account:create", "account:update",
		"deal:list", "deal:create", "deal:update"} {
		if !ops[want] {
			t.Fatalf("expected operation %q", want)
		}
	}
}

func TestZohoCRMBaseURLParam(t *testing.T) {
	def := ZohoCRM().Build()
	hasBaseURL := false
	for _, p := range def.Params {
		if p.Name == "baseUrl" {
			hasBaseURL = true
			if p.Default != "https://www.zohoapis.com/crm/v7" {
				t.Fatalf("expected default base URL, got %v", p.Default)
			}
			break
		}
	}
	if !hasBaseURL {
		t.Fatal("expected baseUrl param for data center override")
	}
}

func TestZohoCRMCredentialType(t *testing.T) {
	def := ZohoCRM().Build()
	found := false
	for _, c := range def.Credentials {
		if c == "zohoCrmApi" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected zohoCrmApi credential type")
	}
}
