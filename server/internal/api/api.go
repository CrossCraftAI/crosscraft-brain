// Package api is the HTTP surface: the same REST + SSE contract the existing
// frontend (apps/studio/lib/client.ts) already calls, served by Go.
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/engine"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/id"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/registry"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/store"
)

// Server bundles the dependencies the handlers need.
type Server struct {
	reg   *registry.Registry
	store *store.Store
	eng   *engine.Engine
}

// NewRouter wires the /api routes.
func NewRouter(reg *registry.Registry, st *store.Store, eng *engine.Engine) http.Handler {
	s := &Server{reg: reg, store: st, eng: eng}
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(cors)

	r.Route("/api", func(r chi.Router) {
		r.Get("/nodes", s.nodes)

		r.Get("/workflows", s.listWorkflows)
		r.Post("/workflows", s.createWorkflow)
		r.Get("/workflows/{id}", s.getWorkflow)
		r.Put("/workflows/{id}", s.saveWorkflow)
		r.Post("/workflows/{id}/run", s.run)

		r.Get("/executions", s.listExecutions)
		r.Get("/executions/{id}", s.getExecution)
		r.Get("/executions/{id}/stream", s.stream)

		r.Post("/resume/{id}", s.resume)
		r.Post("/webhook/{path}", s.webhook)

		r.Get("/credentials", s.listCredentials)
		r.Post("/credentials", s.createCredential)
		r.Delete("/credentials/{id}", s.deleteCredential)

		r.Post("/copilot", s.copilot)
	})
	return r
}

// ---- node catalog ----------------------------------------------------------

func (s *Server) nodes(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.reg.Descriptors())
}

// ---- workflows -------------------------------------------------------------

func (s *Server) listWorkflows(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.ListWorkflows(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) createWorkflow(w http.ResponseWriter, r *http.Request) {
	var body schema.Workflow
	_ = readJSON(r, &body)
	if body.ID == "" {
		body.ID = id.New()
	}
	if body.Name == "" {
		body.Name = "Untitled workflow"
	}
	if body.Nodes == nil {
		body.Nodes = []schema.WFNode{}
	}
	if body.Edges == nil {
		body.Edges = []schema.WFEdge{}
	}
	if body.Settings == nil {
		body.Settings = map[string]any{}
	}
	if err := s.store.SaveWorkflow(r.Context(), &body); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, body)
}

func (s *Server) getWorkflow(w http.ResponseWriter, r *http.Request) {
	wf, err := s.store.LoadWorkflow(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if wf == nil {
		writeErr(w, http.StatusNotFound, fmt.Errorf("workflow not found"))
		return
	}
	writeJSON(w, http.StatusOK, wf)
}

func (s *Server) saveWorkflow(w http.ResponseWriter, r *http.Request) {
	var wf schema.Workflow
	if err := readJSON(r, &wf); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	wf.ID = chi.URLParam(r, "id")
	if wf.Nodes == nil {
		wf.Nodes = []schema.WFNode{}
	}
	if wf.Edges == nil {
		wf.Edges = []schema.WFEdge{}
	}
	if err := s.store.SaveWorkflow(r.Context(), &wf); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, wf)
}

// ---- run / resume / webhook ------------------------------------------------

func (s *Server) run(w http.ResponseWriter, r *http.Request) {
	wf, err := s.store.LoadWorkflow(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if wf == nil {
		writeErr(w, http.StatusNotFound, fmt.Errorf("workflow not found"))
		return
	}
	res, err := s.eng.Run(r.Context(), wf, []schema.Item{{JSON: bodyObject(r)}})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) resume(w http.ResponseWriter, r *http.Request) {
	res, err := s.eng.Resume(r.Context(), chi.URLParam(r, "id"), []schema.Item{{JSON: bodyObject(r)}})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) webhook(w http.ResponseWriter, r *http.Request) {
	path := chi.URLParam(r, "path")
	wfs, err := s.store.ListActiveWorkflows(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	var target *schema.Workflow
	for i := range wfs {
		for _, n := range wfs[i].Nodes {
			if n.Type == "core.webhookTrigger" {
				if p, _ := n.Params["path"].(string); p == path {
					target = &wfs[i]
					break
				}
			}
		}
		if target != nil {
			break
		}
	}
	if target == nil {
		writeErr(w, http.StatusNotFound, fmt.Errorf("no active workflow for webhook path %q", path))
		return
	}
	res, err := s.eng.Run(r.Context(), target, []schema.Item{{JSON: bodyObject(r)}})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if res.Respond != nil {
		status := res.Respond.Status
		if status == 0 {
			status = http.StatusOK
		}
		writeJSON(w, status, res.Respond.Body)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// ---- executions / monitoring ----------------------------------------------

func (s *Server) listExecutions(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.ListExecutions(r.Context(), r.URL.Query().Get("workflowId"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) getExecution(w http.ResponseWriter, r *http.Request) {
	eid := chi.URLParam(r, "id")
	st, err := s.store.GetExecutionStatus(r.Context(), eid)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if !st.Found {
		writeErr(w, http.StatusNotFound, fmt.Errorf("execution not found"))
		return
	}
	steps, err := s.store.GetExecutionSteps(r.Context(), eid)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": st.Status, "waitingNodeId": st.WaitingNodeID, "steps": steps})
}

// stream is the SSE live monitor: poll status + steps until the run finishes.
// Faithful port of apps/studio/app/api/executions/[id]/stream/route.ts.
func (s *Server) stream(w http.ResponseWriter, r *http.Request) {
	eid := chi.URLParam(r, "id")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")

	ctx := r.Context()
	send := func(v any) {
		b, _ := json.Marshal(v)
		fmt.Fprintf(w, "data: %s\n\n", b)
		flusher.Flush()
	}

	for i := 0; i < 600; i++ {
		select {
		case <-ctx.Done():
			return
		default:
		}
		st, err := s.store.GetExecutionStatus(ctx, eid)
		if err != nil || !st.Found {
			send(map[string]any{"error": "not found"})
			return
		}
		steps, _ := s.store.GetExecutionSteps(ctx, eid)
		send(map[string]any{"status": st.Status, "waitingNodeId": st.WaitingNodeID, "steps": steps})
		if st.Status == "success" || st.Status == "error" {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(700 * time.Millisecond):
		}
	}
}

// ---- credentials -----------------------------------------------------------

func (s *Server) listCredentials(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.ListCredentials(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) createCredential(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Type string         `json:"type"`
		Name string         `json:"name"`
		Data map[string]any `json:"data"`
	}
	if err := readJSON(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	row, err := s.store.CreateCredential(r.Context(), body.Type, body.Name, body.Data)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Server) deleteCredential(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteCredential(r.Context(), chi.URLParam(r, "id")); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// copilot is a Phase 1 feature; stubbed so the UI degrades gracefully.
func (s *Server) copilot(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ops":     []any{},
		"message": "Copilot is not implemented in the Go backend yet.",
	})
}

// ---- helpers ---------------------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]any{"error": err.Error()})
}

func readJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// bodyObject decodes the request body as a JSON object, tolerating empty bodies.
func bodyObject(r *http.Request) map[string]any {
	var m map[string]any
	_ = json.NewDecoder(r.Body).Decode(&m)
	if m == nil {
		m = map[string]any{}
	}
	return m
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
