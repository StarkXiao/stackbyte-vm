package api
import (
	"context"
	"fmt"
	"net/http"
	"strings"
)
func (h *Handler) createSession(w http.ResponseWriter, r *http.Request) {
	request, ok := h.readSource(w, r)
	if !ok { return }
	snapshot, err := h.sessions.Create(request.Source)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "debug_create_failed", "debug", err)
		return
	}
	writeJSON(w, http.StatusCreated, snapshot)
}
func (h *Handler) debugSession(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/debug/sessions/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusNotFound, "not_found", "request", fmt.Errorf("session id is required"))
		return
	}
	id := parts[0]
	if r.Method == http.MethodDelete && len(parts) == 1 {
		if !h.sessions.Delete(id) {
			writeError(w, http.StatusNotFound, "not_found", "debug", fmt.Errorf("debug session %q not found", id))
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost || len(parts) != 2 {
		writeError(w, http.StatusNotFound, "not_found", "request", fmt.Errorf("debug action not found"))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), h.cfg.ExecutionTime)
	defer cancel()
	var (
		snapshot any
		err      error
	)
	switch parts[1] {
	case "step": snapshot, err = h.sessions.Step(ctx, id)
	case "continue": snapshot, err = h.sessions.Continue(ctx, id)
	default: writeError(w, http.StatusNotFound, "not_found", "request", fmt.Errorf("unknown debug action %q", parts[1]))
		return
	}
	if err != nil {
		writeError(w, http.StatusNotFound, "debug_failed", "debug", err)
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}
