package google

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

func slidesOauthCtx(params map[string]any, client *http.Client, called *bool) *schema.ExecContext {
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
// Presentation get
// ---------------------------------------------------------------------------

func TestSlidesGetPresentation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/v1/presentations/pres1" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"presentationId": "pres1",
				"title":          "My Presentation",
				"revisionId":     "rev1",
				"pageSize": map[string]any{
					"width":  map[string]any{"magnitude": float64(9144000), "unit": "EMU"},
					"height": map[string]any{"magnitude": float64(5143500), "unit": "EMU"},
				},
				"slides": []any{
					map[string]any{"objectId": "slide1"},
					map[string]any{"objectId": "slide2", "slideProperties": map[string]any{"layoutObjectId": "layout1"}},
				},
			})
			return
		}
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	def := SlidesNode(srv.URL)
	called := false
	ctx := slidesOauthCtx(map[string]any{
		"operation":      "presentation:get",
		"presentationId": "pres1",
		"credential":     "c1",
	}, srv.Client(), &called)

	res, err := def.Execute(ctx)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	out := res.Outputs["main"]
	if len(out) != 1 {
		t.Fatalf("expected 1 presentation, got %d", len(out))
	}
	if out[0].JSON["presentationId"] != "pres1" {
		t.Fatalf("expected presentationId pres1, got %v", out[0].JSON["presentationId"])
	}
	if out[0].JSON["title"] != "My Presentation" {
		t.Fatalf("expected title 'My Presentation', got %v", out[0].JSON["title"])
	}
	slides, _ := out[0].JSON["slides"].([]map[string]any)
	if len(slides) != 2 {
		t.Fatalf("expected 2 slides, got %d", len(slides))
	}
}

// ---------------------------------------------------------------------------
// Presentation create
// ---------------------------------------------------------------------------

func TestSlidesCreatePresentation(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/presentations" {
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			w.Header().Set("Content-Type", "application/json")
			resp := map[string]any{
				"presentationId": "pres-new",
				"title":          gotBody["title"],
				"revisionId":     "rev1",
				"slides":         []any{},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	def := SlidesNode(srv.URL)
	called := false
	ctx := slidesOauthCtx(map[string]any{
		"operation":  "presentation:create",
		"credential": "c1",
		"body":       map[string]any{"title": "New Pres"},
	}, srv.Client(), &called)

	res, err := def.Execute(ctx)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if gotBody == nil {
		t.Fatal("expected a request body")
	}
	if title, _ := gotBody["title"].(string); title != "New Pres" {
		t.Fatalf("expected title 'New Pres', got %v", gotBody)
	}
	out := res.Outputs["main"]
	if len(out) != 1 || out[0].JSON["presentationId"] != "pres-new" {
		t.Fatalf("unexpected create result: %+v", out)
	}
}

// ---------------------------------------------------------------------------
// Presentation update (batchUpdate)
// ---------------------------------------------------------------------------

func TestSlidesUpdatePresentation(t *testing.T) {
	var gotMethod string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		if r.Method == http.MethodPost && r.URL.Path == "/v1/presentations/pres1:batchUpdate" {
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"presentationId": "pres1",
				"replies":        []any{map[string]any{}},
			})
			return
		}
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	def := SlidesNode(srv.URL)
	called := false
	ctx := slidesOauthCtx(map[string]any{
		"operation":      "presentation:update",
		"presentationId": "pres1",
		"credential":     "c1",
		"body": map[string]any{
			"requests": []any{
				map[string]any{
					"replaceAllText": map[string]any{
						"containsText": map[string]any{"text": "{{name}}"},
						"replaceText":  "John",
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
// Get page thumbnail
// ---------------------------------------------------------------------------

func TestSlidesGetThumbnail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/v1/presentations/pres1/pages/slide1/thumbnail" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"contentUrl": "https://example.com/thumb.png",
				"width":      float64(800),
				"height":     float64(600),
			})
			return
		}
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	def := SlidesNode(srv.URL)
	called := false
	ctx := slidesOauthCtx(map[string]any{
		"operation":      "presentation:getThumbnail",
		"presentationId": "pres1",
		"pageObjectId":   "slide1",
		"credential":     "c1",
	}, srv.Client(), &called)

	res, err := def.Execute(ctx)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	out := res.Outputs["main"]
	if len(out) != 1 {
		t.Fatalf("expected 1 thumbnail result, got %d", len(out))
	}
	if out[0].JSON["contentUrl"] != "https://example.com/thumb.png" {
		t.Fatalf("expected contentUrl, got %v", out[0].JSON["contentUrl"])
	}
	w := out[0].JSON["width"]
	if w != int64(800) && w != float64(800) {
		t.Fatalf("expected width 800, got %v (%T)", out[0].JSON["width"], out[0].JSON["width"])
	}
}

// ---------------------------------------------------------------------------
// Operation registration
// ---------------------------------------------------------------------------

func TestSlidesNodeIncludesAllOps(t *testing.T) {
	def := SlidesNode("https://example.test/")
	ops := map[string]bool{}
	for _, p := range def.Params {
		if p.Name == "operation" {
			for _, opt := range p.Options {
				ops[opt.Value] = true
			}
		}
	}

	wantOps := []string{"presentation:get", "presentation:create", "presentation:update", "presentation:getThumbnail"}
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

func TestSlidesGetRequiresPresentationId(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "should not be called", http.StatusInternalServerError)
	}))
	defer srv.Close()

	def := SlidesNode(srv.URL)
	called := false
	ctx := slidesOauthCtx(map[string]any{
		"operation":  "presentation:get",
		"credential": "c1",
	}, srv.Client(), &called)

	_, err := def.Execute(ctx)
	if err == nil {
		t.Fatal("expected error for missing presentationId, got nil")
	}
}

func TestSlidesCreateRequiresTitle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "should not be called", http.StatusInternalServerError)
	}))
	defer srv.Close()

	def := SlidesNode(srv.URL)
	called := false
	ctx := slidesOauthCtx(map[string]any{
		"operation":  "presentation:create",
		"credential": "c1",
		"body":       map[string]any{"notTitle": "x"},
	}, srv.Client(), &called)

	_, err := def.Execute(ctx)
	if err == nil {
		t.Fatal("expected error for missing body.title, got nil")
	}
}

func TestSlidesThumbnailRequiresPageObjectId(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "should not be called", http.StatusInternalServerError)
	}))
	defer srv.Close()

	def := SlidesNode(srv.URL)
	called := false
	ctx := slidesOauthCtx(map[string]any{
		"operation":      "presentation:getThumbnail",
		"presentationId": "pres1",
		"credential":     "c1",
	}, srv.Client(), &called)

	_, err := def.Execute(ctx)
	if err == nil {
		t.Fatal("expected error for missing pageObjectId, got nil")
	}
}

var _ = schema.ExecContext{}
