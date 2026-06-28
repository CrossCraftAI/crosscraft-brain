package google

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

func formsOauthCtx(params map[string]any, client *http.Client, called *bool) *schema.ExecContext {
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
// Form get
// ---------------------------------------------------------------------------

func TestFormsGetForm(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/v1/forms/form1" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"formId":       "form1",
				"revisionId":   "rev1",
				"responderUri": "https://docs.google.com/forms/d/form1/viewform",
				"info": map[string]any{
					"title":         "Customer Survey",
					"description":   "Please fill out this survey",
					"documentTitle": "Customer Survey Form",
				},
				"items": []any{
					map[string]any{
						"itemId": "q1",
						"title":  "Your name",
						"questionItem": map[string]any{
							"question": map[string]any{
								"questionId": "q1",
								"required":   true,
							},
						},
					},
					map[string]any{
						"itemId": "q2",
						"title":  "Feedback",
						"questionItem": map[string]any{
							"question": map[string]any{
								"questionId": "q2",
								"required":   false,
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

	def := FormsNode(srv.URL)
	called := false
	ctx := formsOauthCtx(map[string]any{
		"operation":  "form:get",
		"formId":     "form1",
		"credential": "c1",
	}, srv.Client(), &called)

	res, err := def.Execute(ctx)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	out := res.Outputs["main"]
	if len(out) != 1 {
		t.Fatalf("expected 1 form, got %d", len(out))
	}
	f := out[0].JSON
	if f["formId"] != "form1" {
		t.Fatalf("expected formId form1, got %v", f["formId"])
	}
	info, _ := f["info"].(map[string]any)
	if info["title"] != "Customer Survey" {
		t.Fatalf("expected title 'Customer Survey', got %v", info["title"])
	}
	items, _ := f["items"].([]map[string]any)
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}

// ---------------------------------------------------------------------------
// Response list
// ---------------------------------------------------------------------------

func TestFormsListResponses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/v1/forms/form1/responses" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"responses": []any{
					map[string]any{
						"responseId":        "r1",
						"createTime":        "2025-06-01T10:00:00Z",
						"lastSubmittedTime": "2025-06-01T10:05:00Z",
						"respondentEmail":   "alice@example.com",
						"answers": map[string]any{
							"q1": map[string]any{
								"questionId": "q1",
								"textAnswers": map[string]any{
									"answers": []any{map[string]any{"value": "Alice"}},
								},
							},
						},
					},
					map[string]any{
						"responseId": "r2",
						"createTime": "2025-06-01T11:00:00Z",
						"answers":    map[string]any{},
					},
				},
			})
			return
		}
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	def := FormsNode(srv.URL)
	called := false
	ctx := formsOauthCtx(map[string]any{
		"operation":  "response:list",
		"formId":     "form1",
		"credential": "c1",
	}, srv.Client(), &called)

	res, err := def.Execute(ctx)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	out := res.Outputs["main"]
	if len(out) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(out))
	}
	if out[0].JSON["responseId"] != "r1" {
		t.Fatalf("expected first response r1, got %v", out[0].JSON["responseId"])
	}
}

// ---------------------------------------------------------------------------
// Response get
// ---------------------------------------------------------------------------

func TestFormsGetResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/v1/forms/form1/responses/r1" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"responseId":        "r1",
				"createTime":        "2025-06-01T10:00:00Z",
				"lastSubmittedTime": "2025-06-01T10:05:00Z",
				"respondentEmail":   "john@example.com",
				"answers": map[string]any{
					"q1": map[string]any{
						"questionId": "q1",
						"textAnswers": map[string]any{
							"answers": []any{map[string]any{"value": "John"}},
						},
					},
					"q2": map[string]any{
						"questionId": "q2",
						"textAnswers": map[string]any{
							"answers": []any{map[string]any{"value": "Great product!"}},
						},
					},
				},
			})
			return
		}
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	def := FormsNode(srv.URL)
	called := false
	ctx := formsOauthCtx(map[string]any{
		"operation":  "response:get",
		"formId":     "form1",
		"responseId": "r1",
		"credential": "c1",
	}, srv.Client(), &called)

	res, err := def.Execute(ctx)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	out := res.Outputs["main"]
	if len(out) != 1 {
		t.Fatalf("expected 1 response, got %d", len(out))
	}
	r := out[0].JSON
	if r["responseId"] != "r1" || r["respondentEmail"] != "john@example.com" {
		t.Fatalf("unexpected response: %+v", r)
	}
	answers, _ := r["answers"].(map[string]any)
	if len(answers) != 2 {
		t.Fatalf("expected 2 answers, got %d", len(answers))
	}
}

// ---------------------------------------------------------------------------
// Trigger: new response
// ---------------------------------------------------------------------------

func TestFormsTriggerNewResponse(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && r.URL.Path == "/v1/forms/form1/responses" {
			if callCount == 1 {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"responses": []any{
						map[string]any{"responseId": "r1", "createTime": "2025-06-01T10:00:00Z"},
					},
				})
			} else {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"responses": []any{
						map[string]any{"responseId": "r1", "createTime": "2025-06-01T10:00:00Z"},
						map[string]any{"responseId": "r2", "createTime": "2025-06-01T11:00:00Z"},
					},
				})
			}
			return
		}
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	def := FormsNode(srv.URL)
	called := false
	ctx := formsOauthCtx(map[string]any{
		"operation":  "trigger:newResponse",
		"formId":     "form1",
		"credential": "c1",
	}, srv.Client(), &called)
	ctx.State = map[string]any{}

	// Poll 1 — one new response.
	res1, err := def.Execute(ctx)
	if err != nil {
		t.Fatalf("poll 1: %v", err)
	}
	if n := len(res1.Outputs["main"]); n != 1 {
		t.Fatalf("poll 1: expected 1 new response, got %d", n)
	}

	// Reset rate-limit for second poll.
	delete(ctx.State, "forms:lastPoll:form1:"+srv.URL)

	// Poll 2 — one more new response (r2).
	res2, err := def.Execute(ctx)
	if err != nil {
		t.Fatalf("poll 2: %v", err)
	}
	if n := len(res2.Outputs["main"]); n != 1 {
		t.Fatalf("poll 2: expected 1 new response, got %d", n)
	}
	if res2.Outputs["main"][0].JSON["responseId"] != "r2" {
		t.Fatalf("expected response r2, got %v", res2.Outputs["main"][0].JSON["responseId"])
	}
}

// ---------------------------------------------------------------------------
// Operation registration
// ---------------------------------------------------------------------------

func TestFormsNodeIncludesAllOps(t *testing.T) {
	def := FormsNode("https://example.test/")
	ops := map[string]bool{}
	for _, p := range def.Params {
		if p.Name == "operation" {
			for _, opt := range p.Options {
				ops[opt.Value] = true
			}
		}
	}

	wantOps := []string{"form:get", "form:list", "response:list", "response:get", "trigger:newResponse"}
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

func TestFormsGetRequiresFormId(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "should not be called", http.StatusInternalServerError)
	}))
	defer srv.Close()

	def := FormsNode(srv.URL)
	called := false
	ctx := formsOauthCtx(map[string]any{
		"operation":  "form:get",
		"credential": "c1",
	}, srv.Client(), &called)

	_, err := def.Execute(ctx)
	if err == nil {
		t.Fatal("expected error for missing formId, got nil")
	}
}

func TestFormsGetResponseRequiresResponseId(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "should not be called", http.StatusInternalServerError)
	}))
	defer srv.Close()

	def := FormsNode(srv.URL)
	called := false
	ctx := formsOauthCtx(map[string]any{
		"operation":  "response:get",
		"formId":     "form1",
		"credential": "c1",
	}, srv.Client(), &called)

	_, err := def.Execute(ctx)
	if err == nil {
		t.Fatal("expected error for missing responseId, got nil")
	}
}

var _ = schema.ExecContext{}
