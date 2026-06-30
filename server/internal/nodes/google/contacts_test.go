package google

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

func contactsOauthCtx(params map[string]any, client *http.Client, called *bool) *schema.ExecContext {
	return &schema.ExecContext{
		Params:   params,
		RawParam: func(n string) any { return params[n] },
		AuthorizedClient: func(string) (*http.Client, error) {
			*called = true
			return client, nil
		},
		Log: func(string, any) {},
	}
}

// ---------------------------------------------------------------------------
// Contact list
// ---------------------------------------------------------------------------

func TestContactsList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/v1/people/me/connections") {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"connections": []any{
					map[string]any{
						"resourceName": "people/c1",
						"names":         []any{map[string]any{"givenName": "John", "familyName": "Doe", "displayName": "John Doe"}},
						"emailAddresses": []any{map[string]any{"value": "john@example.com"}},
					},
					map[string]any{
						"resourceName": "people/c2",
						"names":         []any{map[string]any{"givenName": "Jane", "familyName": "Smith", "displayName": "Jane Smith"}},
						"phoneNumbers":  []any{map[string]any{"value": "+1234567890"}},
					},
				},
			})
			return
		}
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	def := ContactsNode(srv.URL)
	called := false
	ctx := contactsOauthCtx(map[string]any{
		"operation":  "contact:list",
		"credential": "c1",
	}, srv.Client(), &called)

	res, err := def.Execute(ctx)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	out := res.Outputs["main"]
	if len(out) != 2 {
		t.Fatalf("expected 2 contacts, got %d", len(out))
	}
	if out[0].JSON["resourceName"] != "people/c1" {
		t.Fatalf("expected first contact people/c1, got %v", out[0].JSON["resourceName"])
	}
}

// ---------------------------------------------------------------------------
// Contact get
// ---------------------------------------------------------------------------

func TestContactsGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/v1/people/c1" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"resourceName": "people/c1",
				"names":         []any{map[string]any{"givenName": "John", "familyName": "Doe", "displayName": "John Doe"}},
				"emailAddresses": []any{
					map[string]any{"value": "john@example.com", "type": "work"},
					map[string]any{"value": "john.doe@gmail.com", "type": "home"},
				},
				"phoneNumbers": []any{map[string]any{"value": "+1234567890", "type": "mobile"}},
				"organizations": []any{map[string]any{"name": "Acme Inc", "title": "Engineer"}},
			})
			return
		}
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	def := ContactsNode(srv.URL)
	called := false
	ctx := contactsOauthCtx(map[string]any{
		"operation":  "contact:get",
		"contactId":  "c1",
		"credential": "c1",
	}, srv.Client(), &called)

	res, err := def.Execute(ctx)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	out := res.Outputs["main"]
	if len(out) != 1 {
		t.Fatalf("expected 1 contact, got %d", len(out))
	}
	c := out[0].JSON
	if c["resourceName"] != "people/c1" {
		t.Fatalf("expected resourceName people/c1, got %v", c["resourceName"])
	}
	emails, _ := c["emailAddresses"].([]map[string]any)
	if len(emails) != 2 {
		t.Fatalf("expected 2 emails, got %d", len(emails))
	}
}

// ---------------------------------------------------------------------------
// Contact create
// ---------------------------------------------------------------------------

func TestContactsCreate(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/people:createContact" {
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			w.Header().Set("Content-Type", "application/json")
			resp := map[string]any{
				"resourceName": "people/c-new",
				"names":         gotBody["names"],
				"emailAddresses": gotBody["emailAddresses"],
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	def := ContactsNode(srv.URL)
	called := false
	ctx := contactsOauthCtx(map[string]any{
		"operation":  "contact:create",
		"credential": "c1",
		"body": map[string]any{
			"names": []any{
				map[string]any{"givenName": "Alice", "familyName": "Wonderland"},
			},
			"emailAddresses": []any{
				map[string]any{"value": "alice@example.com"},
			},
		},
	}, srv.Client(), &called)

	res, err := def.Execute(ctx)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if gotBody == nil {
		t.Fatal("expected a request body")
	}
	names, _ := gotBody["names"].([]any)
	if len(names) != 1 {
		t.Fatalf("expected 1 name, got %v", gotBody)
	}
	out := res.Outputs["main"]
	if len(out) != 1 || out[0].JSON["resourceName"] != "people/c-new" {
		t.Fatalf("unexpected create result: %+v", out)
	}
}

// ---------------------------------------------------------------------------
// Contact update
// ---------------------------------------------------------------------------

func TestContactsUpdate(t *testing.T) {
	var gotMethod string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		if r.Method == http.MethodPatch && r.URL.Path == "/v1/people/c1:updateContact" {
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			w.Header().Set("Content-Type", "application/json")
			resp := map[string]any{
				"resourceName": "people/c1",
				"names":         gotBody["names"],
				"emailAddresses": gotBody["emailAddresses"],
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	def := ContactsNode(srv.URL)
	called := false
	ctx := contactsOauthCtx(map[string]any{
		"operation":  "contact:update",
		"contactId":  "c1",
		"credential": "c1",
		"body": map[string]any{
			"names": []any{
				map[string]any{"givenName": "Bob", "familyName": "Updated"},
			},
		},
	}, srv.Client(), &called)

	res, err := def.Execute(ctx)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if gotMethod != http.MethodPatch {
		t.Fatalf("expected PATCH for update, got %s", gotMethod)
	}
	out := res.Outputs["main"]
	if len(out) != 1 || out[0].JSON["resourceName"] != "people/c1" {
		t.Fatalf("unexpected update result: %+v", out)
	}
}

// ---------------------------------------------------------------------------
// Contact delete
// ---------------------------------------------------------------------------

func TestContactsDelete(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		if r.Method == http.MethodDelete && r.URL.Path == "/v1/people/c1:deleteContact" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{})
			return
		}
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	def := ContactsNode(srv.URL)
	called := false
	ctx := contactsOauthCtx(map[string]any{
		"operation":  "contact:delete",
		"contactId":  "c1",
		"credential": "c1",
	}, srv.Client(), &called)

	res, err := def.Execute(ctx)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Fatalf("expected DELETE, got %s", gotMethod)
	}
	if res.Outputs["main"][0].JSON["deleted"] != true {
		t.Fatal("expected deleted=true")
	}
}

// ---------------------------------------------------------------------------
// Contact get with full resource name (people/ prefix)
// ---------------------------------------------------------------------------

func TestContactsGetWithFullName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/v1/people/c2" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"resourceName": "people/c2",
				"names":         []any{map[string]any{"displayName": "Test User"}},
			})
			return
		}
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	def := ContactsNode(srv.URL)
	called := false
	// Provide the full resource name — should not double-prefix.
	ctx := contactsOauthCtx(map[string]any{
		"operation":  "contact:get",
		"contactId":  "people/c2",
		"credential": "c1",
	}, srv.Client(), &called)

	res, err := def.Execute(ctx)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	out := res.Outputs["main"]
	if len(out) != 1 {
		t.Fatalf("expected 1 contact, got %d", len(out))
	}
}

// ---------------------------------------------------------------------------
// Operation registration
// ---------------------------------------------------------------------------

func TestContactsNodeIncludesAllOps(t *testing.T) {
	def := ContactsNode("https://example.test/")
	ops := map[string]bool{}
	for _, p := range def.Params {
		if p.Name == "operation" {
			for _, opt := range p.Options {
				ops[opt.Value] = true
			}
		}
	}

	wantOps := []string{"contact:list", "contact:get", "contact:create", "contact:update", "contact:delete"}
	for _, want := range wantOps {
		if !ops[want] {
			t.Fatalf("expected operation %q to be registered", want)
		}
	}
	if len(ops) != len(wantOps) {
		t.Fatalf("expected %d operations, got %d (%v)", len(wantOps), len(ops), ops)
	}
}

// ---------------------------------------------------------------------------
// Error paths
// ---------------------------------------------------------------------------

func TestContactsGetRequiresContactId(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "should not be called", http.StatusInternalServerError)
	}))
	defer srv.Close()

	def := ContactsNode(srv.URL)
	called := false
	ctx := contactsOauthCtx(map[string]any{
		"operation":  "contact:get",
		"credential": "c1",
	}, srv.Client(), &called)

	_, err := def.Execute(ctx)
	if err == nil {
		t.Fatal("expected error for missing contactId, got nil")
	}
}

func TestContactsDeleteRequiresContactId(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "should not be called", http.StatusInternalServerError)
	}))
	defer srv.Close()

	def := ContactsNode(srv.URL)
	called := false
	ctx := contactsOauthCtx(map[string]any{
		"operation":  "contact:delete",
		"credential": "c1",
	}, srv.Client(), &called)

	_, err := def.Execute(ctx)
	if err == nil {
		t.Fatal("expected error for missing contactId, got nil")
	}
}

var _ = schema.ExecContext{}
