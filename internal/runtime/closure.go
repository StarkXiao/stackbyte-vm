package runtime
import "fmt"
type Upvalue struct {
	stack  *Stack
	index  int
	closed Value
	open   bool
}
func NewOpenUpvalue(stack *Stack, index int) *Upvalue { return &Upvalue{stack: stack, index: index, open: true} }
func (u *Upvalue) Get() (Value, error) {
	if u == nil { return Nil(), fmt.Errorf("nil upvalue") }
	if u.open { return u.stack.At(u.index) }
	return u.closed, nil
}
func (u *Upvalue) Set(value Value) error {
	if u == nil { return fmt.Errorf("nil upvalue") }
	if u.open { return u.stack.Set(u.index, value) }
	u.closed = value
	return nil
}
func (u *Upvalue) Close() error {
	if u == nil || !u.open { return nil }
	value, err := u.stack.At(u.index)
	if err != nil { return err }
	u.closed = value
	u.stack = nil
	u.open = false
	return nil
}
func (u *Upvalue) IsOpen() bool { return u != nil && u.open }
func (u *Upvalue) Index() int {
	if u == nil || !u.open { return -1 }
	return u.index
}
type Closure struct {
	Function *Function
	Upvalues []*Upvalue
}
func NewClosure(function *Function) *Closure {
	count := 0
	if function != nil { count = len(function.Upvalues) }
	return &Closure{Function: function, Upvalues: make([]*Upvalue, count)}
}
func (c *Closure) SetUpvalue(index int, upvalue *Upvalue) error {
	if c == nil || index < 0 || index >= len(c.Upvalues) { return fmt.Errorf("upvalue index %d is out of range", index) }
	if upvalue == nil { return fmt.Errorf("upvalue %d is nil", index) }
	c.Upvalues[index] = upvalue
	return nil
}
func (c *Closure) GetUpvalue(index int) (*Upvalue, error) {
	if c == nil || index < 0 || index >= len(c.Upvalues) { return nil, fmt.Errorf("upvalue index %d is out of range", index) }
	if c.Upvalues[index] == nil { return nil, fmt.Errorf("upvalue %d has not been initialized", index) }
	return c.Upvalues[index], nil
}
