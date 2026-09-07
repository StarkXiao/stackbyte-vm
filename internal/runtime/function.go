package runtime
import (
	"fmt"
	"stackbyte-vm/internal/bytecode"
)
type UpvalueSpec struct {
	Index   byte
	IsLocal bool
}
type Function struct {
	Name      string
	Arity     int
	Chunk     *bytecode.Chunk
	Upvalues  []UpvalueSpec
	MaxLocals int
}
func NewFunction(name string) *Function {
	return &Function{
		Name:     name,
		Chunk:    bytecode.NewChunk(),
		Upvalues: make([]UpvalueSpec, 0, 4),
	}
}
func (f *Function) DisplayName() string {
	if f == nil || f.Name == "" { return "<script>" }
	return f.Name
}
func (f *Function) AddUpvalue(index byte, isLocal bool) (byte, error) {
	for i, spec := range f.Upvalues {
		if spec.Index == index && spec.IsLocal == isLocal { return byte(i), nil }
	}
	if len(f.Upvalues) >= 256 { return 0, fmt.Errorf("function %s captures too many values", f.DisplayName()) }
	f.Upvalues = append(f.Upvalues, UpvalueSpec{Index: index, IsLocal: isLocal})
	return byte(len(f.Upvalues) - 1), nil
}
func (f *Function) Validate() error {
	if f == nil { return fmt.Errorf("function is nil") }
	if f.Arity < 0 || f.Arity > 255 { return fmt.Errorf("function %s has invalid arity %d", f.DisplayName(), f.Arity) }
	if err := bytecode.Validate(f.Chunk); err != nil { return fmt.Errorf("function %s: %w", f.DisplayName(), err) }
	for _, constant := range f.Chunk.Constants {
		if child, ok := constant.(*Function); ok {
			if err := child.Validate(); err != nil { return err }
		}
	}
	return nil
}
type NativeFunc func(args []Value) (Value, error)
type Native struct {
	Name     string
	MinArity int
	MaxArity int
	Call     NativeFunc
}
func (n *Native) Invoke(args []Value) (Value, error) {
	if n == nil || n.Call == nil { return Nil(), fmt.Errorf("native function is not callable") }
	if len(args) < n.MinArity || len(args) > n.MaxArity { return Nil(), fmt.Errorf("%s expects %d..%d arguments, got %d", n.Name, n.MinArity, n.MaxArity, len(args)) }
	return n.Call(args)
}
