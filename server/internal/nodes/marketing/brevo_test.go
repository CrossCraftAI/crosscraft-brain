package marketing

import (
	"testing"
)

func TestBrevoNode(t *testing.T) {
	def := Brevo().Build()
	if def.Type != "marketing.brevo" {
		t.Fatalf("unexpected type: %s", def.Type)
	}
	ops := collectOps(def)
	if len(ops) < 10 {
		t.Fatalf("expected at least 10 ops, got %d", len(ops))
	}
	for _, want := range []string{"contact:list", "contact:create", "contact:update", "contact:delete",
		"email:send", "email:sendTemplate",
		"campaign:list", "campaign:create", "campaign:send",
		"list:list", "list:create"} {
		if !ops[want] {
			t.Fatalf("expected operation %q", want)
		}
	}
}

func TestBrevoAuthHeader(t *testing.T) {
	def := Brevo().Build()
	found := false
	for _, c := range def.Credentials {
		if c == "brevoApi" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected brevoApi credential type")
	}
}

func TestBrevoSendTemplateHasTemplateParam(t *testing.T) {
	def := Brevo().Build()
	hasTemplate := false
	for _, p := range def.Params {
		if p.Name == "templateId" {
			hasTemplate = true
			break
		}
	}
	if !hasTemplate {
		t.Fatal("expected templateId param for email:sendTemplate operation")
	}
}

func TestMarketingNodeCount(t *testing.T) {
	nodes := Nodes()
	if len(nodes) != 3 {
		t.Fatalf("expected 3 marketing nodes, got %d", len(nodes))
	}
}
