package vm
import (
	"bytes"
	"context"
	"fmt"
	"stackbyte-vm/internal/bytecode"
	vmruntime "stackbyte-vm/internal/runtime"
	"time"
)
type Limits struct {
	MaxStack       int
	MaxFrames      int
	MaxSteps       int
	MaxOutputBytes int
}
func DefaultLimits() Limits { return Limits{MaxStack: 4096, MaxFrames: 64, MaxSteps: 1_000_000, MaxOutputBytes: 256 << 10} }
type Stats struct {
	Instructions int           `json:"instructions"`
	MaxStack     int           `json:"max_stack"`
	MaxFrames    int           `json:"max_frames"`
	Duration     time.Duration `json:"duration"`
	OutputBytes  int           `json:"output_bytes"`
}
type Result struct {
	Value  vmruntime.Value
	Output string
	Stats  Stats
}
type TraceFrame struct {
	Function string `json:"function"`
	Line     int    `json:"line"`
}
type RuntimeError struct {
	Message string       `json:"message"`
	Trace   []TraceFrame `json:"trace,omitempty"`
}
func (e *RuntimeError) Error() string { return e.Message }
type VM struct {
	limits  Limits
	stack   *vmruntime.Stack
	frames  []vmruntime.Frame
	globals map[string]vmruntime.Value
	open    map[int]*vmruntime.Upvalue
	output  bytes.Buffer
	stats   Stats
	started time.Time
	done    bool
	result  vmruntime.Value
}
func New(limits Limits) *VM {
	if limits.MaxStack <= 0 { limits = DefaultLimits() }
	machine := &VM{
		limits:  limits,
		stack:   vmruntime.NewStack(limits.MaxStack),
		globals: make(map[string]vmruntime.Value),
		open:    make(map[int]*vmruntime.Upvalue),
	}
	machine.installNatives()
	return machine
}
func (v *VM) Prepare(function *vmruntime.Function) error {
	if err := function.Validate(); err != nil { return err }
	v.Reset()
	closure := vmruntime.NewClosure(function)
	if err := v.stack.Push(vmruntime.ClosureValue(closure)); err != nil { return err }
	if err := v.callClosure(closure, 0); err != nil { return err }
	v.started = time.Now()
	return nil
}
func (v *VM) Execute(ctx context.Context, function *vmruntime.Function) (Result, error) {
	if err := v.Prepare(function); err != nil { return Result{}, err }
	for !v.done {
		if err := v.Step(ctx); err != nil { return Result{}, err }
	}
	v.stats.Duration = time.Since(v.started)
	v.stats.MaxStack = v.stack.Peak()
	v.stats.OutputBytes = v.output.Len()
	return Result{Value: v.result, Output: v.output.String(), Stats: v.stats}, nil
}
func (v *VM) Reset() {
	v.stack.Reset()
	v.frames = v.frames[:0]
	v.open = make(map[int]*vmruntime.Upvalue)
	v.output.Reset()
	v.stats = Stats{}
	v.done = false
	v.result = vmruntime.Nil()
}
func (v *VM) Done() bool     { return v.done }
func (v *VM) Output() string { return v.output.String() }
func (v *VM) StackSnapshot() []string {
	values := v.stack.Snapshot()
	result := make([]string, len(values))
	for i, value := range values {
		result[i] = value.String()
	}
	return result
}
func (v *VM) FrameSnapshot() []TraceFrame {
	result := make([]TraceFrame, 0, len(v.frames))
	for i := len(v.frames) - 1; i >= 0; i-- {
		frame := v.frames[i]
		line := frame.Closure.Function.Chunk.Line(max(frame.IP-1, 0))
		result = append(result, TraceFrame{Function: frame.Name(), Line: line})
	}
	return result
}
func (v *VM) CurrentInstruction() (int, string, int) {
	if len(v.frames) == 0 { return -1, "DONE", 0 }
	frame := v.frames[len(v.frames)-1]
	chunk := frame.Closure.Function.Chunk
	if frame.IP >= len(chunk.Code) { return frame.IP, "EOF", chunk.Line(max(frame.IP-1, 0)) }
	return frame.IP, bytecode.OpCode(chunk.Code[frame.IP]).String(), chunk.Line(frame.IP)
}
func (v *VM) runtimeError(format string, args ...any) error { return &RuntimeError{Message: fmt.Sprintf(format, args...), Trace: v.FrameSnapshot()} }
