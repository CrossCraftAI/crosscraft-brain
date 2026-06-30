package microsoft

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

func oauthCtx(params map[string]any, client *http.Client) *schema.ExecContext {
	return &schema.ExecContext{
		Params:           params,
		RawParam:         func(n string) any { return params[n] },
		AuthorizedClient: func(string) (*http.Client, error) { return client, nil },
		Log:              func(string, any) {},
	}
}

// ---------------------------------------------------------------------------
// Outlook
// ---------------------------------------------------------------------------

func TestOutlookListMessages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/me/messages" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"value": []any{map[string]any{"id": "m1"}, map[string]any{"id": "m2"}},
			})
			return
		}
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	def := Outlook(srv.URL).Build()
	res, err := def.Execute(oauthCtx(map[string]any{"operation": "message:list", "credential": "c1"}, srv.Client()))
	if err != nil {
		t.Fatal(err)
	}
	if out := res.Outputs["main"]; len(out) != 2 || out[0].JSON["id"] != "m1" {
		t.Fatalf("messages: %+v", res.Outputs["main"])
	}
}

func TestOutlookSendMessage(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	def := Outlook(srv.URL).Build()
	ctx := oauthCtx(map[string]any{
		"operation": "message:send", "credential": "c1",
		"body": map[string]any{
			"message": map[string]any{
				"subject": "Test",
				"body":    map[string]any{"contentType": "Text", "content": "Hello"},
				"toRecipients": []any{map[string]any{"emailAddress": map[string]any{"address": "user@example.com"}}},
			},
		},
	}, srv.Client())
	res, err := def.Execute(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/me/sendMail" {
		t.Fatalf("path: %s", gotPath)
	}
	if len(res.Outputs["main"]) == 0 {
		t.Fatal("expected non-empty output")
	}
}

func TestOutlookReply(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	def := Outlook(srv.URL).Build()
	ctx := oauthCtx(map[string]any{
		"operation": "message:reply", "credential": "c1", "messageId": "msg1",
		"body": map[string]any{"comment": "Thanks for the update."},
	}, srv.Client())
	_, err := def.Execute(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/me/messages/msg1/reply" {
		t.Fatalf("path: %s", gotPath)
	}
}

func TestOutlookMoveMessage(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "msg1"})
	}))
	defer srv.Close()

	def := Outlook(srv.URL).Build()
	ctx := oauthCtx(map[string]any{
		"operation": "message:move", "credential": "c1", "messageId": "msg1",
		"body": map[string]any{"destinationId": "inbox"},
	}, srv.Client())
	res, err := def.Execute(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/me/messages/msg1/move" {
		t.Fatalf("path: %s", gotPath)
	}
	if res.Outputs["main"][0].JSON["id"] != "msg1" {
		t.Fatalf("unexpected output: %+v", res.Outputs["main"])
	}
}

func TestOutlookCreateFolder(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "folder1", "displayName": "TestFolder"})
	}))
	defer srv.Close()

	def := Outlook(srv.URL).Build()
	ctx := oauthCtx(map[string]any{
		"operation": "folder:create", "credential": "c1",
		"body": map[string]any{"displayName": "TestFolder"},
	}, srv.Client())
	res, err := def.Execute(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/me/mailFolders" {
		t.Fatalf("path: %s", gotPath)
	}
	if res.Outputs["main"][0].JSON["displayName"] != "TestFolder" {
		t.Fatalf("unexpected: %+v", res.Outputs["main"])
	}
}

func TestOutlookDraftList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The $ character is URL-encoded as %24 in query parameter names.
		if r.Method == http.MethodGet && r.URL.Query().Get("$filter") == "isDraft eq true" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"value": []any{map[string]any{"id": "d1", "isDraft": true}},
			})
			return
		}
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	def := Outlook(srv.URL).Build()
	res, err := def.Execute(oauthCtx(map[string]any{"operation": "draft:list", "credential": "c1"}, srv.Client()))
	if err != nil {
		t.Fatal(err)
	}
	if out := res.Outputs["main"]; len(out) != 1 || out[0].JSON["id"] != "d1" {
		t.Fatalf("drafts: %+v", res.Outputs["main"])
	}
}

// ---------------------------------------------------------------------------
// Teams
// ---------------------------------------------------------------------------

func TestTeamsSendChannelMessage(t *testing.T) {
	var gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "msg1"})
	}))
	defer srv.Close()

	def := Teams(srv.URL).Build()
	ctx := oauthCtx(map[string]any{
		"operation": "channelMessage:send", "teamId": "T1", "channelId": "C1",
		"body":      map[string]any{"body": map[string]any{"content": "hi"}},
		"credential": "c1",
	}, srv.Client())
	res, err := def.Execute(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/teams/T1/channels/C1/messages" {
		t.Fatalf("path: %s", gotPath)
	}
	if gotBody != `{"body":{"content":"hi"}}` {
		t.Fatalf("body: %s", gotBody)
	}
	if res.Outputs["main"][0].JSON["id"] != "msg1" {
		t.Fatalf("output: %+v", res.Outputs["main"])
	}
}

func TestTeamsCreateChannel(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "ch1", "displayName": "General"})
	}))
	defer srv.Close()

	def := Teams(srv.URL).Build()
	ctx := oauthCtx(map[string]any{
		"operation": "channel:create", "credential": "c1", "teamId": "T1",
		"body": map[string]any{"displayName": "General", "description": "General chat"},
	}, srv.Client())
	res, err := def.Execute(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost || gotPath != "/teams/T1/channels" {
		t.Fatalf("unexpected: %s %s", gotMethod, gotPath)
	}
	if res.Outputs["main"][0].JSON["id"] != "ch1" {
		t.Fatalf("unexpected: %+v", res.Outputs["main"])
	}
}

// ---------------------------------------------------------------------------
// ToDo
// ---------------------------------------------------------------------------

func TestToDoCreateList(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "l1", "displayName": "Shopping"})
	}))
	defer srv.Close()

	def := ToDo(srv.URL).Build()
	ctx := oauthCtx(map[string]any{
		"operation": "list:create", "credential": "c1",
		"body": map[string]any{"displayName": "Shopping"},
	}, srv.Client())
	res, err := def.Execute(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/me/todo/lists" {
		t.Fatalf("path: %s", gotPath)
	}
	if res.Outputs["main"][0].JSON["displayName"] != "Shopping" {
		t.Fatalf("unexpected: %+v", res.Outputs["main"])
	}
}

func TestToDoUpdateTask(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "t1", "title": "Done", "status": "completed"})
	}))
	defer srv.Close()

	def := ToDo(srv.URL).Build()
	ctx := oauthCtx(map[string]any{
		"operation": "task:update", "credential": "c1", "listId": "l1", "taskId": "t1",
		"body": map[string]any{"status": "completed"},
	}, srv.Client())
	res, err := def.Execute(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPatch || gotPath != "/me/todo/lists/l1/tasks/t1" {
		t.Fatalf("unexpected: %s %s", gotMethod, gotPath)
	}
	if res.Outputs["main"][0].JSON["status"] != "completed" {
		t.Fatalf("unexpected: %+v", res.Outputs["main"])
	}
}

// ---------------------------------------------------------------------------
// Excel
// ---------------------------------------------------------------------------

func TestExcelGetRange(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"values": []any{[]any{"Name", "Age"}, []any{"Alice", 30}},
		})
	}))
	defer srv.Close()

	def := Excel(srv.URL).Build()
	ctx := oauthCtx(map[string]any{
		"operation": "range:get", "credential": "c1", "itemId": "wb1", "worksheetId": "Sheet1", "address": "A1:B2",
	}, srv.Client())
	res, err := def.Execute(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/me/drive/items/wb1/workbook/worksheets/Sheet1/range(address='A1:B2')" {
		t.Fatalf("path: %s", gotPath)
	}
	if len(res.Outputs["main"]) != 1 || res.Outputs["main"][0].JSON["values"] == nil {
		t.Fatalf("unexpected: %+v", res.Outputs["main"])
	}
}

// ---------------------------------------------------------------------------
// OneDrive
// ---------------------------------------------------------------------------

func TestOneDriveSearch(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"value": []any{map[string]any{"id": "f1", "name": "report.docx"}},
		})
	}))
	defer srv.Close()

	def := OneDrive(srv.URL).Build()
	ctx := oauthCtx(map[string]any{
		"operation": "item:search", "credential": "c1", "q": "report",
	}, srv.Client())
	res, err := def.Execute(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/me/drive/root/search(q='report')" {
		t.Fatalf("path: %s", gotPath)
	}
	if out := res.Outputs["main"]; len(out) != 1 || out[0].JSON["name"] != "report.docx" {
		t.Fatalf("search results: %+v", res.Outputs["main"])
	}
}

func TestOneDriveShare(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"value": []any{map[string]any{"grantedTo": map[string]any{"user": map[string]any{"email": "bob@example.com"}}}},
		})
	}))
	defer srv.Close()

	def := OneDrive(srv.URL).Build()
	ctx := oauthCtx(map[string]any{
		"operation": "item:share", "credential": "c1", "itemId": "f1",
		"body": map[string]any{
			"recipients":    []any{map[string]any{"email": "bob@example.com"}},
			"message":       "Here is the file",
			"requireSignIn": true,
			"sendInvitation": true,
			"roles":         []any{"read"},
		},
	}, srv.Client())
	res, err := def.Execute(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/me/drive/items/f1/invite" {
		t.Fatalf("path: %s", gotPath)
	}
	if len(res.Outputs["main"]) != 1 {
		t.Fatalf("unexpected: %+v", res.Outputs["main"])
	}
}

// ---------------------------------------------------------------------------
// Calendar
// ---------------------------------------------------------------------------

func TestCalUpdateEvent(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "ev1", "subject": "Updated"})
	}))
	defer srv.Close()

	def := Calendar(srv.URL).Build()
	ctx := oauthCtx(map[string]any{
		"operation": "event:update", "credential": "c1", "eventId": "ev1",
		"body": map[string]any{"subject": "Updated"},
	}, srv.Client())
	res, err := def.Execute(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPatch || gotPath != "/me/events/ev1" {
		t.Fatalf("unexpected: %s %s", gotMethod, gotPath)
	}
	if res.Outputs["main"][0].JSON["subject"] != "Updated" {
		t.Fatalf("unexpected: %+v", res.Outputs["main"])
	}
}

func TestCalListCalendars(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/me/calendars" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"value": []any{map[string]any{"id": "cal1", "name": "Calendar"}, map[string]any{"id": "cal2", "name": "Work"}},
			})
			return
		}
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	def := Calendar(srv.URL).Build()
	res, err := def.Execute(oauthCtx(map[string]any{"operation": "calendar:list", "credential": "c1"}, srv.Client()))
	if err != nil {
		t.Fatal(err)
	}
	if out := res.Outputs["main"]; len(out) != 2 || out[0].JSON["id"] != "cal1" {
		t.Fatalf("calendars: %+v", res.Outputs["main"])
	}
}

// ---------------------------------------------------------------------------
// SharePoint
// ---------------------------------------------------------------------------

func TestSharePointGetSite(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "contoso.sharepoint.com,site1", "displayName": "Contoso"})
	}))
	defer srv.Close()

	def := SharePoint(srv.URL).Build()
	res, err := def.Execute(oauthCtx(map[string]any{
		"operation": "site:get", "credential": "c1", "siteId": "contoso.sharepoint.com,site1",
	}, srv.Client()))
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/sites/contoso.sharepoint.com,site1" {
		t.Fatalf("path: %s", gotPath)
	}
	if res.Outputs["main"][0].JSON["displayName"] != "Contoso" {
		t.Fatalf("unexpected: %+v", res.Outputs["main"])
	}
}

func TestSharePointListItems(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/sites/s1/lists/l1/items" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"value": []any{
					map[string]any{"id": "i1", "fields": map[string]any{"Title": "Item 1"}},
					map[string]any{"id": "i2", "fields": map[string]any{"Title": "Item 2"}},
				},
			})
			return
		}
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	def := SharePoint(srv.URL).Build()
	res, err := def.Execute(oauthCtx(map[string]any{
		"operation": "listItem:list", "credential": "c1", "siteId": "s1", "listId": "l1",
	}, srv.Client()))
	if err != nil {
		t.Fatal(err)
	}
	if out := res.Outputs["main"]; len(out) != 2 || out[0].JSON["id"] != "i1" {
		t.Fatalf("items: %+v", res.Outputs["main"])
	}
}

func TestSharePointCreateListItem(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "i3", "fields": map[string]any{"Title": "New Item"}})
	}))
	defer srv.Close()

	def := SharePoint(srv.URL).Build()
	ctx := oauthCtx(map[string]any{
		"operation": "listItem:create", "credential": "c1", "siteId": "s1", "listId": "l1",
		"body": map[string]any{"fields": map[string]any{"Title": "New Item"}},
	}, srv.Client())
	res, err := def.Execute(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost || gotPath != "/sites/s1/lists/l1/items" {
		t.Fatalf("unexpected: %s %s", gotMethod, gotPath)
	}
	if res.Outputs["main"][0].JSON["id"] != "i3" {
		t.Fatalf("unexpected: %+v", res.Outputs["main"])
	}
}

// ---------------------------------------------------------------------------
// OneNote
// ---------------------------------------------------------------------------

func TestOneNoteListNotebooks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/me/onenote/notebooks" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"value": []any{
					map[string]any{"id": "nb1", "displayName": "Personal"},
					map[string]any{"id": "nb2", "displayName": "Work"},
				},
			})
			return
		}
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	def := OneNote(srv.URL).Build()
	res, err := def.Execute(oauthCtx(map[string]any{"operation": "notebook:list", "credential": "c1"}, srv.Client()))
	if err != nil {
		t.Fatal(err)
	}
	if out := res.Outputs["main"]; len(out) != 2 || out[0].JSON["displayName"] != "Personal" {
		t.Fatalf("notebooks: %+v", res.Outputs["main"])
	}
}

func TestOneNoteCreatePage(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "pg1", "title": "Meeting Notes"})
	}))
	defer srv.Close()

	def := OneNote(srv.URL).Build()
	ctx := oauthCtx(map[string]any{
		"operation": "page:create", "credential": "c1", "sectionId": "sec1",
		"body": map[string]any{"title": "Meeting Notes", "content": "<p>Notes here</p>"},
	}, srv.Client())
	res, err := def.Execute(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost || gotPath != "/me/onenote/sections/sec1/pages" {
		t.Fatalf("unexpected: %s %s", gotMethod, gotPath)
	}
	if res.Outputs["main"][0].JSON["title"] != "Meeting Notes" {
		t.Fatalf("unexpected: %+v", res.Outputs["main"])
	}
}

// ---------------------------------------------------------------------------
// Graph (generic)
// ---------------------------------------------------------------------------

func TestGraphGetCall(t *testing.T) {
	var gotURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		// Return data directly (no "value" wrapper) since the raw node has no ItemsPath.
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "u1", "displayName": "Alice"})
	}))
	defer srv.Close()

	def := Graph(srv.URL).Build()
	ctx := oauthCtx(map[string]any{
		"operation": "raw:get", "credential": "c1",
		"rawUrl": srv.URL + "/me/people",
	}, srv.Client())
	res, err := def.Execute(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if gotURL != "/me/people" {
		t.Fatalf("path: %s", gotURL)
	}
	out := res.Outputs["main"]
	if len(out) != 1 || out[0].JSON["id"] != "u1" {
		t.Fatalf("results: %+v", res.Outputs["main"])
	}
}

func TestGraphPostCall(t *testing.T) {
	var gotURL, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "new", "status": "ok"})
	}))
	defer srv.Close()

	def := Graph(srv.URL).Build()
	ctx := oauthCtx(map[string]any{
		"operation": "raw:post", "credential": "c1",
		"rawUrl": srv.URL + "/me/messages",
		"body":   map[string]any{"subject": "Test"},
	}, srv.Client())
	res, err := def.Execute(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost || gotURL != "/me/messages" {
		t.Fatalf("unexpected: %s %s", gotMethod, gotURL)
	}
	if res.Outputs["main"][0].JSON["id"] != "new" {
		t.Fatalf("unexpected: %+v", res.Outputs["main"])
	}
}

// ---------------------------------------------------------------------------
// Node registration
// ---------------------------------------------------------------------------

func TestNodesRegisterable(t *testing.T) {
	nodes := Nodes()
	if len(nodes) != 9 {
		t.Fatalf("expected 9 microsoft nodes, got %d", len(nodes))
	}
	for _, n := range nodes {
		if n.Params[0].Type != "credential" {
			t.Fatalf("%s: first param should be credential, got %+v", n.Type, n.Params[0])
		}
		hasOp := false
		for _, p := range n.Params {
			if p.Name == "operation" {
				hasOp = true
				break
			}
		}
		if !hasOp {
			t.Fatalf("%s: missing operation param", n.Type)
		}
	}
}

var _ = schema.ExecContext{}
