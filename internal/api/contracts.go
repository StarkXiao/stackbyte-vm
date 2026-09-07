package api
import (
	"encoding/json"
	"net/http"
	"stackbyte-vm/internal/compiler"
	"stackbyte-vm/internal/vm"
)
type sourceRequest struct {
	Source string `json:"source"`
}
type compileResponse struct {
	Disassembly string                `json:"disassembly"`
	Constants   int                   `json:"constants"`
	CodeBytes   int                   `json:"code_bytes"`
	Diagnostics []compiler.Diagnostic `json:"diagnostics"`
}
type runResponse struct {
	Output string        `json:"output"`
	Value  string        `json:"value"`
	Stats  responseStats `json:"stats"`
}
type responseStats struct {
	Instructions int     `json:"instructions"`
	MaxStack     int     `json:"max_stack"`
	MaxFrames    int     `json:"max_frames"`
	DurationMS   float64 `json:"duration_ms"`
	OutputBytes  int     `json:"output_bytes"`
}
func statsResponse(stats vm.Stats) responseStats {
	return responseStats{
		Instructions: stats.Instructions, MaxStack: stats.MaxStack, MaxFrames: stats.MaxFrames,
		DurationMS: float64(stats.Duration.Microseconds()) / 1000, OutputBytes: stats.OutputBytes,
	}
}
type apiError struct {
	Code        string                `json:"code"`
	Message     string                `json:"message"`
	Phase       string                `json:"phase"`
	Diagnostics []compiler.Diagnostic `json:"diagnostics,omitempty"`
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, status int, code, phase string, err error) {
	result := apiError{Code: code, Message: err.Error(), Phase: phase}
	if compileErr, ok := err.(*compiler.CompileError); ok { result.Diagnostics = compileErr.Diagnostics }
	writeJSON(w, status, result)
}
