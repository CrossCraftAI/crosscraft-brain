package google

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

func docsOauthCtx(params map[string]any, client *http.Client, called *bool) *schema.ExecContext {
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
// Document get
// ---------------------------------------------------------------------------

func TestDocsGetDocument(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/v1/documents/doc1" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"documentId": "doc1",
				"title":      "My Document",
				"revisionId": "abc123",
				"body": map[string]any{
					"content": []any{
						map[string]any{
							"paragraph": map[string]any{
								"elements": []any{
									map[string]any{"textRun": map[string]any{"content": "Hello World"}},
								},
							},
						},
					},
				},
			})
			return
		}
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	def := DocsNode(srv.URL)
	called := false
	ctx := docsOauthCtx(map[string]any{
		"operation":  "document:get",
		"documentId": "doc1",
		"credential": "c1",
	}, srv.Client(), &called)

	res, err := def.Execute(ctx)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	out := res.Outputs["main"]
	if len(out) != 1 {
		t.Fatalf("expected 1 document, got %d", len(out))
	}
	if out[0].JSON["documentId"] != "doc1" {
		t.Fatalf("expected documentId doc1, got %v", out[0].JSON["documentId"])
	}
	if out[0].JSON["title"] != "My Document" {
		t.Fatalf("expected title 'My Document', got %v", out[0].JSON["title"])
	}
}

// ---------------------------------------------------------------------------
// Document create
// ---------------------------------------------------------------------------

func TestDocsCreateDocument(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/documents" {
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			w.Header().Set("Content-Type", "application/json")
			resp := map[string]any{
				"documentId": "doc-new",
				"title":      gotBody["title"],
				"revisionId": "rev1",
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	def := DocsNode(srv.URL)
	called := false
	ctx := docsOauthCtx(map[string]any{
		"operation":  "document:create",
		"credential": "c1",
		"body":       map[string]any{"title": "New Doc"},
	}, srv.Client(), &called)

	res, err := def.Execute(ctx)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if gotBody == nil {
		t.Fatal("expected a request body")
	}
	if title, _ := gotBody["title"].(string); title != "New Doc" {
		t.Fatalf("expected title 'New Doc', got %v", gotBody)
	}
	out := res.Outputs["main"]
	if len(out) != 1 || out[0].JSON["documentId"] != "doc-new" {
		t.Fatalf("unexpected create result: %+v", out)
	}
}

// ---------------------------------------------------------------------------
// Document update (batchUpdate)
// ---------------------------------------------------------------------------

func TestDocsUpdateDocument(t *testing.T) {
	var gotMethod string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		if r.Method == http.MethodPost && r.URL.Path == "/v1/documents/doc1:batchUpdate" {
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"documentId": "doc1",
				"replies":    []any{map[string]any{}},
				"writeControl": map[string]any{
					"requiredRevisionId": "rev2",
				},
			})
			return
		}
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	def := DocsNode(srv.URL)
	called := false
	ctx := docsOauthCtx(map[string]any{
		"operation":  "document:update",
		"documentId": "doc1",
		"credential": "c1",
		"body": map[string]any{
			"requests": []any{
				map[string]any{
					"insertText": map[string]any{
						"location": map[string]any{"index": float64(1)},
						"text":     "Hello",
					},
				},
			},
		},
	}, srv.Client(), &called)

	res, err := def.Execute(ctx)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("expected POST for batchUpdate, got %s", gotMethod)
	}
	if gotBody == nil || len(gotBody["requests"].([]any)) == 0 {
		t.Fatalf("expected batch update request, got %v", gotBody)
	}
	out := res.Outputs["main"]
	if len(out) != 1 || out[0].JSON["updated"] != true {
		t.Fatalf("unexpected update result: %+v", out)
	}
}

// ---------------------------------------------------------------------------
// Operation registration
// ---------------------------------------------------------------------------

func TestDocsNodeIncludesAllOps(t *testing.T) {
	def := DocsNode("https://example.test/")
	ops := map[string]bool{}
	for _, p := range def.Params {
		if p.Name == "operation" {
			for _, opt := range p.Options {
				ops[opt.Value] = true
			}
		}
	}

	wantOps := []string{"document:get", "document:create", "document:update", "document:delete"}
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

func TestDocsGetRequiresDocumentId(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "should not be called", http.StatusInternalServerError)
	}))
	defer srv.Close()

	def := DocsNode(srv.URL)
	called := false
	ctx := docsOauthCtx(map[string]any{
		"operation":  "document:get",
		"credential": "c1",
	}, srv.Client(), &called)

	_, err := def.Execute(ctx)
	if err == nil {
		t.Fatal("expected error for missing documentId, got nil")
	}
}

func TestDocsCreateRequiresTitle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "should not be called", http.StatusInternalServerError)
	}))
	defer srv.Close()

	def := DocsNode(srv.URL)
	called := false
	ctx := docsOauthCtx(map[string]any{
		"operation":  "document:create",
		"credential": "c1",
		"body":       map[string]any{"notTitle": "x"},
	}, srv.Client(), &called)

	_, err := def.Execute(ctx)
	if err == nil {
		t.Fatal("expected error for missing body.title, got nil")
	}
}

func TestDocsUpdateRequiresDocumentId(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "should not be called", http.StatusInternalServerError)
	}))
	defer srv.Close()

	def := DocsNode(srv.URL)
	called := false
	ctx := docsOauthCtx(map[string]any{
		"operation":  "document:update",
		"credential": "c1",
		"body":       map[string]any{"requests": []any{map[string]any{}}},
	}, srv.Client(), &called)

	_, err := def.Execute(ctx)
	if err == nil {
		t.Fatal("expected error for missing documentId, got nil")
	}
}

func TestDocsUpdateRequiresRequests(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "should not be called", http.StatusInternalServerError)
	}))
	defer srv.Close()

	def := DocsNode(srv.URL)
	called := false
	ctx := docsOauthCtx(map[string]any{
		"operation":  "document:update",
		"documentId": "doc1",
		"credential": "c1",
		"body":       map[string]any{"requests": []any{}},
	}, srv.Client(), &called)

	_, err := def.Execute(ctx)
	if err == nil {
		t.Fatal("expected error for empty requests, got nil")
	}
}

var _ = schema.ExecContext{}
