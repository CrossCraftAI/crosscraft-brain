package protocols

import (
	"testing"
)

func TestSOAPNode(t *testing.T) {
	def := SOAP()
	if def.Type != "protocols.soap" {
		t.Fatalf("unexpected type: %s", def.Type)
	}
	if def.Execute == nil {
		t.Fatal("expected execute function to be set")
	}
	if len(def.Outputs) != 2 {
		t.Fatalf("expected 2 outputs (main + error), got %d", len(def.Outputs))
	}

	ops := collectOps(def)
	for _, want := range []string{"call", "callWithHeaders"} {
		if !ops[want] {
			t.Fatalf("expected operation %q", want)
		}
	}
}

func TestSOAPEnvelopeBuild(t *testing.T) {
	env := buildSOAPEnvelope("1.2", "", "<ns:Test/>")
	if env == "" {
		t.Fatal("expected non-empty SOAP envelope")
	}
	if !contains(env, "xmlns:soap=") {
		t.Fatal("expected SOAP namespace in envelope")
	}
	if !contains(env, "<soap:Body>") {
		t.Fatal("expected SOAP body in envelope")
	}
	if !contains(env, "<ns:Test/>") {
		t.Fatal("expected body content in envelope")
	}
}

func TestSOAPEnvelopeVersion11(t *testing.T) {
	env := buildSOAPEnvelope("1.1", "", "<test/>")
	if !contains(env, "schemas.xmlsoap.org/soap/envelope/") {
		t.Fatal("expected SOAP 1.1 namespace")
	}
}

func TestSOAPExtractBody(t *testing.T) {
	raw := `<?xml version="1.0"?><soap:Envelope xmlns:soap="http://www.w3.org/2003/05/soap-envelope"><soap:Body><result>ok</result></soap:Body></soap:Envelope>`
	body := extractSOAPBody(raw)
	if !contains(body, "<result>ok</result>") {
		t.Fatalf("expected body content, got: %s", body)
	}
}

func TestSOAPEnvelopeWithWSecurity(t *testing.T) {
	security := `<wsse:Security><wsse:UsernameToken><wsse:Username>admin</wsse:Username></wsse:UsernameToken></wsse:Security>`
	env := buildSOAPEnvelope("1.2", security, "<ns:Test/>")
	if !contains(env, "<soap:Header>") {
		t.Fatal("expected SOAP header in envelope")
	}
	if !contains(env, "UsernameToken") {
		t.Fatal("expected WS-Security content in header")
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
