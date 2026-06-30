package protocols

import (
	"testing"
)

func TestGraphQLNode(t *testing.T) {
	def := GraphQL()
	if def.Type != "protocols.graphql" {
		t.Fatalf("unexpected type: %s", def.Type)
	}
	if def.Execute == nil {
		t.Fatal("expected execute function to be set")
	}
	if len(def.Params) < 5 {
		t.Fatalf("expected at least 5 params, got %d", len(def.Params))
	}

	ops := collectOps(def)
	for _, want := range []string{"query", "mutation"} {
		if !ops[want] {
			t.Fatalf("expected operation %q", want)
		}
	}
}

func TestGraphQLHasVariablesAndHeaders(t *testing.T) {
	def := GraphQL()
	hasVars := false
	hasHeaders := false
	for _, p := range def.Params {
		if p.Name == "variables" {
			hasVars = true
		}
		if p.Name == "headers" {
			hasHeaders = true
		}
	}
	if !hasVars {
		t.Fatal("expected variables param")
	}
	if !hasHeaders {
		t.Fatal("expected headers param")
	}
}
