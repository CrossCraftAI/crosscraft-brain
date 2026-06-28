package google

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

func tasksOauthCtx(params map[string]any, client *http.Client, called *bool) *schema.ExecContext {
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
// Task list list
// ---------------------------------------------------------------------------

func TestTasksListTasklists(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/tasks/v1/users/@me/lists") {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []any{
					map[string]any{"id": "tl1", "title": "Personal", "updated": "2025-01-01T00:00:00Z"},
					map[string]any{"id": "tl2", "title": "Work", "updated": "2025-01-02T00:00:00Z"},
				},
			})
			return
		}
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	def := TasksNode(srv.URL)
	called := false
	ctx := tasksOauthCtx(map[string]any{
		"operation":  "tasklist:list",
		"credential": "c1",
	}, srv.Client(), &called)

	res, err := def.Execute(ctx)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	out := res.Outputs["main"]
	if len(out) != 2 {
		t.Fatalf("expected 2 task lists, got %d", len(out))
	}
	if out[0].JSON["id"] != "tl1" {
		t.Fatalf("expected first tasklist tl1, got %v", out[0].JSON["id"])
	}
}

// ---------------------------------------------------------------------------
// Task list create
// ---------------------------------------------------------------------------

func TestTasksCreateTasklist(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/tasks/v1/users/@me/lists" {
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			w.Header().Set("Content-Type", "application/json")
			resp := map[string]any{
				"id":    "tl-new",
				"title": gotBody["title"],
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	def := TasksNode(srv.URL)
	called := false
	ctx := tasksOauthCtx(map[string]any{
		"operation":  "tasklist:create",
		"credential": "c1",
		"body":       map[string]any{"title": "New List"},
	}, srv.Client(), &called)

	res, err := def.Execute(ctx)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if gotBody == nil {
		t.Fatal("expected a request body")
	}
	out := res.Outputs["main"]
	if len(out) != 1 || out[0].JSON["id"] != "tl-new" {
		t.Fatalf("unexpected create result: %+v", out)
	}
}

// ---------------------------------------------------------------------------
// Task list delete
// ---------------------------------------------------------------------------

func TestTasksDeleteTasklist(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		if r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/tasks/v1/users/@me/lists/tl1") {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	def := TasksNode(srv.URL)
	called := false
	ctx := tasksOauthCtx(map[string]any{
		"operation":  "tasklist:delete",
		"tasklistId": "tl1",
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
// Task list
// ---------------------------------------------------------------------------

func TestTasksListTasks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/tasks/v1/lists/tl1/tasks") {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []any{
					map[string]any{
						"id": "t1", "title": "Buy groceries", "status": "needsAction",
						"due": "2025-06-01T00:00:00Z", "notes": "Milk, eggs, bread",
					},
					map[string]any{
						"id": "t2", "title": "Call dentist", "status": "completed",
						"completed": "2025-05-28T10:00:00Z",
					},
				},
			})
			return
		}
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	def := TasksNode(srv.URL)
	called := false
	ctx := tasksOauthCtx(map[string]any{
		"operation":  "task:list",
		"tasklistId": "tl1",
		"credential": "c1",
	}, srv.Client(), &called)

	res, err := def.Execute(ctx)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	out := res.Outputs["main"]
	if len(out) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(out))
	}
	if out[0].JSON["id"] != "t1" || out[0].JSON["title"] != "Buy groceries" {
		t.Fatalf("unexpected first task: %+v", out[0].JSON)
	}
}

// ---------------------------------------------------------------------------
// Task create
// ---------------------------------------------------------------------------

func TestTasksCreateTask(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/tasks/v1/lists/tl1/tasks") {
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			w.Header().Set("Content-Type", "application/json")
			resp := map[string]any{
				"id":     "t-new",
				"title":  gotBody["title"],
				"notes":  gotBody["notes"],
				"due":    gotBody["due"],
				"status": "needsAction",
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	def := TasksNode(srv.URL)
	called := false
	ctx := tasksOauthCtx(map[string]any{
		"operation":  "task:create",
		"tasklistId": "tl1",
		"credential": "c1",
		"body": map[string]any{
			"title": "New Task",
			"notes": "Do something",
			"due":   "2025-07-01T09:00:00Z",
		},
	}, srv.Client(), &called)

	res, err := def.Execute(ctx)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if gotBody == nil {
		t.Fatal("expected a request body")
	}
	if title, _ := gotBody["title"].(string); title != "New Task" {
		t.Fatalf("expected title 'New Task', got %v", gotBody)
	}
	out := res.Outputs["main"]
	if len(out) != 1 || out[0].JSON["id"] != "t-new" {
		t.Fatalf("unexpected create result: %+v", out)
	}
}

// ---------------------------------------------------------------------------
// Task update
// ---------------------------------------------------------------------------

func TestTasksUpdateTask(t *testing.T) {
	var gotMethod string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		if r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/tasks/v1/lists/tl1/tasks/t1") {
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			w.Header().Set("Content-Type", "application/json")
			resp := map[string]any{
				"id":     "t1",
				"title":  gotBody["title"],
				"status": "completed",
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	def := TasksNode(srv.URL)
	called := false
	ctx := tasksOauthCtx(map[string]any{
		"operation":  "task:update",
		"tasklistId": "tl1",
		"taskId":     "t1",
		"credential": "c1",
		"body": map[string]any{
			"title":  "Updated Task",
			"status": "completed",
		},
	}, srv.Client(), &called)

	res, err := def.Execute(ctx)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Fatalf("expected PUT for update, got %s", gotMethod)
	}
	out := res.Outputs["main"]
	if len(out) != 1 || out[0].JSON["title"] != "Updated Task" {
		t.Fatalf("unexpected update result: %+v", out)
	}
}

// ---------------------------------------------------------------------------
// Task delete
// ---------------------------------------------------------------------------

func TestTasksDeleteTask(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		if r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/tasks/v1/lists/tl1/tasks/t1") {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	def := TasksNode(srv.URL)
	called := false
	ctx := tasksOauthCtx(map[string]any{
		"operation":  "task:delete",
		"tasklistId": "tl1",
		"taskId":     "t1",
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
// Operation registration
// ---------------------------------------------------------------------------

func TestTasksNodeIncludesAllOps(t *testing.T) {
	def := TasksNode("https://example.test/")
	ops := map[string]bool{}
	for _, p := range def.Params {
		if p.Name == "operation" {
			for _, opt := range p.Options {
				ops[opt.Value] = true
			}
		}
	}

	wantOps := []string{"tasklist:list", "tasklist:create", "tasklist:delete",
		"task:list", "task:create", "task:update", "task:delete"}
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

func TestTasksCreateTaskRequiresTasklistId(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "should not be called", http.StatusInternalServerError)
	}))
	defer srv.Close()

	def := TasksNode(srv.URL)
	called := false
	ctx := tasksOauthCtx(map[string]any{
		"operation":  "task:create",
		"credential": "c1",
		"body":       map[string]any{"title": "No list"},
	}, srv.Client(), &called)

	_, err := def.Execute(ctx)
	if err == nil {
		t.Fatal("expected error for missing tasklistId, got nil")
	}
}

func TestTasksDeleteRequiresTaskId(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "should not be called", http.StatusInternalServerError)
	}))
	defer srv.Close()

	def := TasksNode(srv.URL)
	called := false
	ctx := tasksOauthCtx(map[string]any{
		"operation":  "task:delete",
		"tasklistId": "tl1",
		"credential": "c1",
	}, srv.Client(), &called)

	_, err := def.Execute(ctx)
	if err == nil {
		t.Fatal("expected error for missing taskId, got nil")
	}
}

func TestTasksUpdateRequiresTasklistId(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "should not be called", http.StatusInternalServerError)
	}))
	defer srv.Close()

	def := TasksNode(srv.URL)
	called := false
	ctx := tasksOauthCtx(map[string]any{
		"operation": "task:update",
		"taskId":    "t1",
		"credential": "c1",
		"body":      map[string]any{"title": "No list"},
	}, srv.Client(), &called)

	_, err := def.Execute(ctx)
	if err == nil {
		t.Fatal("expected error for missing tasklistId, got nil")
	}
}

var _ = schema.ExecContext{}
