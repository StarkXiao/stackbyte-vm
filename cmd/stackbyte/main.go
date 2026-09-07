package main
import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"stackbyte-vm/internal/api"
	"stackbyte-vm/internal/bytecode"
	"stackbyte-vm/internal/compiler"
	"stackbyte-vm/internal/config"
	"stackbyte-vm/internal/vm"
	"strings"
	"syscall"
	"time"
)
var version = "dev"
func main() {
	os.Exit(run(os.Args[1:]))
}
func run(args []string) int {
	if len(args) == 0 {
		usage()
		return 2
	}
	switch args[0] {
	case "run": if len(args) != 2 {
			usage()
			return 2
		}
		return runFile(args[1], false)
	case "check": if len(args) != 2 {
			usage()
			return 2
		}
		return checkFile(args[1])
	case "disasm": if len(args) != 2 {
			usage()
			return 2
		}
		return runFile(args[1], true)
	case "repl": return repl()
	case "serve": return serve()
	case "version", "--version", "-v": fmt.Println("stackbyte", version)
		return 0
	default: fmt.Fprintf(os.Stderr, "unknown command %q\n", args[0])
		usage()
		return 2
	}
}
func readAndCompile(path string) (string, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil { return "", fmt.Errorf("read %s: %w", path, err) }
	return string(data), nil
}
func runFile(path string, disassemble bool) int {
	source, err := readAndCompile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 74
	}
	function, err := compiler.Compile(source)
	if err != nil {
		printCompileError(err)
		return 65
	}
	if disassemble {
		fmt.Print(bytecode.Disassemble(function.DisplayName(), function.Chunk))
		return 0
	}
	result, err := vm.New(vm.DefaultLimits()).Execute(context.Background(), function)
	if err != nil {
		fmt.Fprintln(os.Stderr, "runtime error:", err)
		return 70
	}
	fmt.Print(result.Output)
	return 0
}
func checkFile(path string) int {
	source, err := readAndCompile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 74
	}
	function, err := compiler.Compile(source)
	if err != nil {
		printCompileError(err)
		return 65
	}
	fmt.Printf("OK: %d bytecode bytes, %d constants\n", len(function.Chunk.Code), len(function.Chunk.Constants))
	return 0
}
func repl() int {
	fmt.Println("StackByte VM", version, "- end each statement with ';', Ctrl-D to exit")
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() { break }
		source := strings.TrimSpace(scanner.Text())
		if source == "" { continue }
		function, err := compiler.Compile(source)
		if err != nil {
			printCompileError(err)
			continue
		}
		result, err := vm.New(vm.DefaultLimits()).Execute(context.Background(), function)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		fmt.Print(result.Output)
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 74
	}
	return 0
}
func serve() int {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 74
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	server, err := api.NewServer(cfg, logger)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 74
	}
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-done
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()
	logger.Info("StackByte VM web console", "address", "http://"+cfg.Address)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("server stopped", "error", err)
		return 74
	}
	return 0
}
func printCompileError(err error) {
	var compileErr *compiler.CompileError
	if errors.As(err, &compileErr) {
		for _, item := range compileErr.Diagnostics {
			fmt.Fprintf(os.Stderr, "%s %d:%d: %s\n", item.Phase, item.Line, item.Column, item.Message)
		}
		return
	}
	fmt.Fprintln(os.Stderr, err)
}
func usage() {
	fmt.Fprintln(os.Stderr, "usage: stackbyte <run|check|disasm|repl|serve|version> [file]")
}
