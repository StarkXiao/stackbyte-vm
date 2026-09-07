package api
import (
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"stackbyte-vm/internal/config"
	webassets "stackbyte-vm/web"
	"time"
)
func NewServer(cfg config.Config, logger *slog.Logger) (*http.Server, error) {
	handler := NewHandler(cfg)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/compile", handler.compile)
	mux.HandleFunc("POST /api/v1/run", handler.run)
	mux.HandleFunc("GET /api/v1/examples", handler.examples)
	mux.HandleFunc("GET /healthz", handler.health)
	mux.HandleFunc("POST /api/v1/debug/sessions", handler.createSession)
	mux.HandleFunc("/api/v1/debug/sessions/", handler.debugSession)
	static, err := fs.Sub(webassets.Files, ".")
	if err != nil { return nil, fmt.Errorf("prepare embedded web assets: %w", err) }
	mux.Handle("/", http.FileServer(http.FS(static)))
	return &http.Server{
		Addr: cfg.Address, Handler: requestLog(logger, securityHeaders(mux)),
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second,
		WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second,
	}, nil
}
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'")
		next.ServeHTTP(w, r)
	})
}
func requestLog(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Info("http request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start))
	})
}
