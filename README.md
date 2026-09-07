# StackByte VM

StackByte VM is a compact stack-based bytecode virtual machine written in Go. It includes a scanner, Pratt compiler, bytecode validator, function calls, lexical closures, a debugger, five example programs, and an embedded browser console.

## Run

```sh
go run ./cmd/stackbyte run examples/counter.sb
go run ./cmd/stackbyte disasm examples/factorial.sb
go run ./cmd/stackbyte serve
```

Open <http://127.0.0.1:8080> after starting the server.

## Verify and package

```sh
make check
make build
make package
```

`make package` creates archives for Linux, macOS, and Windows on amd64 and arm64 where applicable. The web UI is embedded in every binary and has no Node.js dependency.

Configuration uses `STACKBYTE_ADDR`, `STACKBYTE_MAX_STACK`, `STACKBYTE_MAX_FRAMES`, `STACKBYTE_MAX_STEPS`, `STACKBYTE_MAX_OUTPUT`, and `STACKBYTE_MAX_SESSIONS`.
