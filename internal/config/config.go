package config
import (
	"fmt"
	"os"
	"strconv"
	"time"
)
type Config struct {
	Address        string
	MaxSourceBytes int64
	MaxStack       int
	MaxFrames      int
	MaxSteps       int
	MaxOutputBytes int
	ExecutionTime  time.Duration
	SessionTTL     time.Duration
	MaxSessions    int
}
func Default() Config {
	return Config{
		Address:        "127.0.0.1:8080",
		MaxSourceBytes: 128 << 10,
		MaxStack:       4096,
		MaxFrames:      64,
		MaxSteps:       1_000_000,
		MaxOutputBytes: 256 << 10,
		ExecutionTime:  3 * time.Second,
		SessionTTL:     10 * time.Minute,
		MaxSessions:    64,
	}
}
func Load() (Config, error) {
	cfg := Default()
	if value := os.Getenv("STACKBYTE_ADDR"); value != "" { cfg.Address = value }
	items := []struct {
		name string
		dst  *int
	}{
		{"STACKBYTE_MAX_STACK", &cfg.MaxStack},
		{"STACKBYTE_MAX_FRAMES", &cfg.MaxFrames},
		{"STACKBYTE_MAX_STEPS", &cfg.MaxSteps},
		{"STACKBYTE_MAX_OUTPUT", &cfg.MaxOutputBytes},
		{"STACKBYTE_MAX_SESSIONS", &cfg.MaxSessions},
	}
	for _, item := range items {
		if value := os.Getenv(item.name); value != "" {
			n, err := strconv.Atoi(value)
			if err != nil || n <= 0 { return Config{}, fmt.Errorf("%s must be a positive integer", item.name) }
			*item.dst = n
		}
	}
	if cfg.MaxFrames > cfg.MaxStack { return Config{}, fmt.Errorf("maximum frames cannot exceed maximum stack") }
	return cfg, nil
}
