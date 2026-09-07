package vm_test

import (
	"context"
	"strings"
	"testing"

	"stackbyte-vm/internal/compiler"
	"stackbyte-vm/internal/vm"
)

func execute(t *testing.T, source string) string {
	t.Helper()
	function, err := compiler.Compile(source)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	result, err := vm.New(vm.DefaultLimits()).Execute(context.Background(), function)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	return result.Output
}

func TestArithmeticControlFlowAndNativeFunctions(t *testing.T) {
	source := `
var total = 0;
var i = 1;
while (i <= 4) { total = total + i; i = i + 1; }
if (total == 10 and len("vm") == 2) print "ok"; else print "bad";
print type(total);
`
	if got := execute(t, source); got != "ok\nnumber\n" {
		t.Fatalf("unexpected output %q", got)
	}
}

func TestRecursiveFunction(t *testing.T) {
	source := `fun fact(n) { if (n <= 1) return 1; return n * fact(n - 1); } print fact(6);`
	if got := execute(t, source); got != "720\n" {
		t.Fatalf("unexpected output %q", got)
	}
}

func TestClosureKeepsStateAfterReturn(t *testing.T) {
	source := `
fun make() { var n = 0; fun next() { n = n + 1; return n; } return next; }
var a = make(); var b = make(); print a(); print a(); print b();
`
	if got := execute(t, source); got != "1\n2\n1\n" {
		t.Fatalf("unexpected output %q", got)
	}
}

func TestRuntimeErrorsAreReturned(t *testing.T) {
	function, err := compiler.Compile("print 1 / 0;")
	if err != nil {
		t.Fatal(err)
	}
	_, err = vm.New(vm.DefaultLimits()).Execute(context.Background(), function)
	if err == nil || !strings.Contains(err.Error(), "division by zero") {
		t.Fatalf("expected division error, got %v", err)
	}
}
