package api
import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"stackbyte-vm/internal/bytecode"
	"stackbyte-vm/internal/compiler"
	"stackbyte-vm/internal/config"
	"stackbyte-vm/internal/debug"
	"stackbyte-vm/internal/vm"
)
type Handler struct {
	cfg      config.Config
	limits   vm.Limits
	sessions *debug.Manager
}
func NewHandler(cfg config.Config) *Handler {
	limits := vm.Limits{
		MaxStack: cfg.MaxStack, MaxFrames: cfg.MaxFrames,
		MaxSteps: cfg.MaxSteps, MaxOutputBytes: cfg.MaxOutputBytes,
	}
	return &Handler{cfg: cfg, limits: limits, sessions: debug.NewManager(limits, cfg.SessionTTL, cfg.MaxSessions)}
}
func (h *Handler) compile(w http.ResponseWriter, r *http.Request) {
	request, ok := h.readSource(w, r)
	if !ok { return }
	function, err := compiler.Compile(request.Source)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "compile_failed", "compile", err)
		return
	}
	writeJSON(w, http.StatusOK, compileResponse{
		Disassembly: bytecode.Disassemble(function.DisplayName(), function.Chunk),
		Constants:   len(function.Chunk.Constants), CodeBytes: len(function.Chunk.Code),
		Diagnostics: []compiler.Diagnostic{},
	})
}
func (h *Handler) run(w http.ResponseWriter, r *http.Request) {
	request, ok := h.readSource(w, r)
	if !ok { return }
	function, err := compiler.Compile(request.Source)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "compile_failed", "compile", err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), h.cfg.ExecutionTime)
	defer cancel()
	result, err := vm.New(h.limits).Execute(ctx, function)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "runtime_failed", "runtime", err)
		return
	}
	writeJSON(w, http.StatusOK, runResponse{Output: result.Output, Value: result.Value.String(), Stats: statsResponse(result.Stats)})
}
func (h *Handler) examples(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, BuiltinExamples())
}
func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "stackbyte-vm"})
}
func (h *Handler) readSource(w http.ResponseWriter, r *http.Request) (sourceRequest, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, h.cfg.MaxSourceBytes)
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var request sourceRequest
	if err := decoder.Decode(&request); err != nil {
		status := http.StatusBadRequest
		if err == io.EOF { err = fmt.Errorf("request body is required") }
		writeError(w, status, "invalid_request", "request", err)
		return sourceRequest{}, false
	}
	if request.Source == "" {
		writeError(w, http.StatusBadRequest, "source_required", "request", fmt.Errorf("source is required"))
		return sourceRequest{}, false
	}
	return request, true
}
type Example struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Source      string `json:"source"`
}
func BuiltinExamples() []Example {
	return []Example{
		{"arithmetic", "Arithmetic and comparison", "print 1 + 2 * 3;\nprint 10 > 4;\n"},
		{"fibonacci", "Iterative Fibonacci", "var a = 0; var b = 1; var i = 0;\nwhile (i < 10) { print a; var n = a + b; a = b; b = n; i = i + 1; }\n"},
		{"factorial", "Recursive function calls", "fun factorial(n) { if (n <= 1) return 1; return n * factorial(n - 1); }\nprint factorial(6);\n"},
		{"counter", "A closure with private state", "fun makeCounter() { var n = 0; fun next() { n = n + 1; return n; } return next; }\nvar counter = makeCounter(); print counter(); print counter(); print counter();\n"},
		{"shared-upvalue", "Two closures sharing one captured value", "fun makePair() { var n = 0; fun add() { n = n + 1; return n; } fun read() { return n; } print add(); print read(); }\nmakePair();\n"},
	}
}
