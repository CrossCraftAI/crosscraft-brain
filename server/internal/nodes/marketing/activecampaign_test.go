package marketing

import (
	"testing"
)

func TestActiveCampaignNode(t *testing.T) {
	def := ActiveCampaign().Build()
	if def.Type != "marketing.activecampaign" {
		t.Fatalf("unexpected type: %s", def.Type)
	}
	ops := collectOps(def)
	if len(ops) < 9 {
		t.Fatalf("expected at least 9 ops, got %d", len(ops))
	}
	for _, want := range []string{"contact:list", "contact:create", "contact:update", "contact:delete",
		"list:list", "list:create",
		"automation:list", "automation:trigger",
		"campaign:list", "campaign:create", "message:send"} {
		if !ops[want] {
			t.Fatalf("expected operation %q", want)
		}
	}
}

func TestActiveCampaignAuthHeader(t *testing.T) {
	def := ActiveCampaign().Build()
	found := false
	for _, c := range def.Credentials {
		if c == "activecampaignApi" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected activecampaignApi credential type")
	}
}

func TestActiveCampaignBaseURLParam(t *testing.T) {
	def := ActiveCampaign().Build()
	hasBaseURL := false
	for _, p := range def.Params {
		if p.Name == "baseUrl" {
			hasBaseURL = true
			break
		}
	}
	if !hasBaseURL {
		t.Fatal("expected baseUrl param for account subdomain override")
	}
}
