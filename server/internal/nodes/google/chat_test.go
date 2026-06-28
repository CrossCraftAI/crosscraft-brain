package google

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

func chatOauthCtx(params map[string]any, client *http.Client, called *bool) *schema.ExecContext {
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
// Space list
// ---------------------------------------------------------------------------

func TestChatListSpaces(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/v1/spaces" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"spaces": []any{
					map[string]any{
						"name":        "spaces/AAA",
						"displayName": "General",
						"type":        "ROOM",
						"threaded":    true,
					},
					map[string]any{
						"name":        "spaces/BBB",
						"displayName": "Engineering",
						"type":        "ROOM",
						"threaded":    true,
						"spaceThreadingState": "THREADED_MESSAGES",
					},
				},
			})
			return
		}
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	def := ChatNode(srv.URL)
	called := false
	ctx := chatOauthCtx(map[string]any{
		"operation":  "space:list",
		"credential": "c1",
	}, srv.Client(), &called)

	res, err := def.Execute(ctx)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	out := res.Outputs["main"]
	if len(out) != 2 {
		t.Fatalf("expected 2 spaces, got %d", len(out))
	}
	if out[0].JSON["name"] != "spaces/AAA" || out[0].JSON["displayName"] != "General" {
		t.Fatalf("unexpected first space: %+v", out[0].JSON)
	}
}

// ---------------------------------------------------------------------------
// Space get
// ---------------------------------------------------------------------------

func TestChatGetSpace(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/v1/spaces/AAA" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"name":        "spaces/AAA",
				"displayName": "General Chat",
				"type":        "ROOM",
				"threaded":    true,
				"spaceThreadingState": "THREADED_MESSAGES",
			})
			return
		}
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	def := ChatNode(srv.URL)
	called := false
	ctx := chatOauthCtx(map[string]any{
		"operation":  "space:get",
		"spaceName":  "AAA",
		"credential": "c1",
	}, srv.Client(), &called)

	res, err := def.Execute(ctx)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	out := res.Outputs["main"]
	if len(out) != 1 {
		t.Fatalf("expected 1 space, got %d", len(out))
	}
	if out[0].JSON["name"] != "spaces/AAA" {
		t.Fatalf("expected space name spaces/AAA, got %v", out[0].JSON["name"])
	}
}

// ---------------------------------------------------------------------------
// Member list
// ---------------------------------------------------------------------------

func TestChatListMembers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/v1/spaces/AAA/members" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"memberships": []any{
					map[string]any{
						"name":  "spaces/AAA/members/1",
						"state": "JOINED",
						"role":  "ROLE_MEMBER",
						"member": map[string]any{
							"name":        "users/1",
							"displayName": "Alice",
							"email":       "alice@example.com",
							"type":        "HUMAN",
						},
					},
					map[string]any{
						"name":  "spaces/AAA/members/2",
						"state": "JOINED",
						"role":  "ROLE_MANAGER",
						"member": map[string]any{
							"name":        "users/2",
							"displayName": "Bob",
							"email":       "bob@example.com",
							"type":        "HUMAN",
						},
					},
				},
			})
			return
		}
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	def := ChatNode(srv.URL)
	called := false
	ctx := chatOauthCtx(map[string]any{
		"operation":  "member:list",
		"spaceName":  "spaces/AAA",
		"credential": "c1",
	}, srv.Client(), &called)

	res, err := def.Execute(ctx)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	out := res.Outputs["main"]
	if len(out) != 2 {
		t.Fatalf("expected 2 members, got %d", len(out))
	}
	m := out[0].JSON
	if m["name"] != "spaces/AAA/members/1" {
		t.Fatalf("unexpected first member: %+v", m)
	}
	member, _ := m["member"].(map[string]any)
	if member["displayName"] != "Alice" {
		t.Fatalf("expected member name Alice, got %v", member["displayName"])
	}
}

// ---------------------------------------------------------------------------
// Message send
// ---------------------------------------------------------------------------

func TestChatSendMessage(t *testing.T) {
	var gotMethod string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		if r.Method == http.MethodPost && r.URL.Path == "/v1/spaces/AAA/messages" {
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"name":       "spaces/AAA/messages/msg1",
				"text":       gotBody["text"],
				"createTime": "2025-06-01T12:00:00Z",
				"sender": map[string]any{
					"name":        "users/1",
					"displayName": "Test Bot",
					"type":        "BOT",
				},
				"space": map[string]any{
					"name":        "spaces/AAA",
					"displayName": "General",
					"type":        "ROOM",
				},
			})
			return
		}
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	def := ChatNode(srv.URL)
	called := false
	ctx := chatOauthCtx(map[string]any{
		"operation":  "message:send",
		"spaceName":  "AAA",
		"text":       "Hello from the bot!",
		"credential": "c1",
	}, srv.Client(), &called)

	res, err := def.Execute(ctx)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("expected POST for send, got %s", gotMethod)
	}
	if text, _ := gotBody["text"].(string); text != "Hello from the bot!" {
		t.Fatalf("expected text in body, got %v", gotBody)
	}
	out := res.Outputs["main"]
	if len(out) != 1 || out[0].JSON["text"] != "Hello from the bot!" {
		t.Fatalf("unexpected send result: %+v", out)
	}
}

// ---------------------------------------------------------------------------
// Space get with full name (spaces/ prefix)
// ---------------------------------------------------------------------------

func TestChatGetSpaceWithFullName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/v1/spaces/BBB" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"name":        "spaces/BBB",
				"displayName": "Engineering",
				"type":        "ROOM",
				"threaded":    true,
			})
			return
		}
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	def := ChatNode(srv.URL)
	called := false
	ctx := chatOauthCtx(map[string]any{
		"operation":  "space:get",
		"spaceName":  "spaces/BBB",
		"credential": "c1",
	}, srv.Client(), &called)

	res, err := def.Execute(ctx)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	out := res.Outputs["main"]
	if len(out) != 1 {
		t.Fatalf("expected 1 space, got %d", len(out))
	}
}

// ---------------------------------------------------------------------------
// Operation registration
// ---------------------------------------------------------------------------

func TestChatNodeIncludesAllOps(t *testing.T) {
	def := ChatNode("https://example.test/")
	ops := map[string]bool{}
	for _, p := range def.Params {
		if p.Name == "operation" {
			for _, opt := range p.Options {
				ops[opt.Value] = true
			}
		}
	}

	wantOps := []string{"space:list", "space:get", "member:list", "message:send"}
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

func TestChatSendRequiresText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "should not be called", http.StatusInternalServerError)
	}))
	defer srv.Close()

	def := ChatNode(srv.URL)
	called := false
	ctx := chatOauthCtx(map[string]any{
		"operation":  "message:send",
		"spaceName":  "AAA",
		"credential": "c1",
	}, srv.Client(), &called)

	_, err := def.Execute(ctx)
	if err == nil {
		t.Fatal("expected error for missing text, got nil")
	}
}

func TestChatGetRequiresSpaceName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "should not be called", http.StatusInternalServerError)
	}))
	defer srv.Close()

	def := ChatNode(srv.URL)
	called := false
	ctx := chatOauthCtx(map[string]any{
		"operation":  "space:get",
		"credential": "c1",
	}, srv.Client(), &called)

	_, err := def.Execute(ctx)
	if err == nil {
		t.Fatal("expected error for missing spaceName, got nil")
	}
}

func TestChatListMembersRequiresSpaceName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "should not be called", http.StatusInternalServerError)
	}))
	defer srv.Close()

	def := ChatNode(srv.URL)
	called := false
	ctx := chatOauthCtx(map[string]any{
		"operation":  "member:list",
		"credential": "c1",
	}, srv.Client(), &called)

	_, err := def.Execute(ctx)
	if err == nil {
		t.Fatal("expected error for missing spaceName, got nil")
	}
}

var _ = schema.ExecContext{}
