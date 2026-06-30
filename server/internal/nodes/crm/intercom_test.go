package crm

import (
	"testing"
)

func TestIntercomNode(t *testing.T) {
	def := Intercom().Build()
	if def.Type != "crm.intercom" {
		t.Fatalf("unexpected type: %s", def.Type)
	}
	ops := collectOps(def)
	if len(ops) < 8 {
		t.Fatalf("expected at least 8 ops, got %d", len(ops))
	}
	for _, want := range []string{"contact:list", "contact:create", "contact:update", "contact:delete",
		"conversation:list", "conversation:get", "conversation:reply",
		"company:list", "company:create"} {
		if !ops[want] {
			t.Fatalf("expected operation %q", want)
		}
	}
}

func TestIntercomAuthHeader(t *testing.T) {
	def := Intercom().Build()
	found := false
	for _, c := range def.Credentials {
		if c == "intercomApi" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected intercomApi credential type")
	}
}

func TestIntercomVersionHeader(t *testing.T) {
	def := Intercom().Build()
	if def.Label != "Intercom" {
		t.Fatalf("expected label Intercom, got %s", def.Label)
	}
}
