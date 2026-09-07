package runtime
import "fmt"
type Stack struct {
	values []Value
	limit  int
	peak   int
}
func NewStack(limit int) *Stack {
	if limit <= 0 { limit = 1 }
	return &Stack{values: make([]Value, 0, min(limit, 256)), limit: limit}
}
func (s *Stack) Push(value Value) error {
	if len(s.values) >= s.limit { return fmt.Errorf("stack overflow: limit is %d", s.limit) }
	s.values = append(s.values, value)
	if len(s.values) > s.peak { s.peak = len(s.values) }
	return nil
}
func (s *Stack) Pop() (Value, error) {
	if len(s.values) == 0 { return Nil(), fmt.Errorf("stack underflow") }
	last := len(s.values) - 1
	value := s.values[last]
	s.values[last] = Nil()
	s.values = s.values[:last]
	return value, nil
}
func (s *Stack) Peek(distance int) (Value, error) { return s.At(len(s.values) - 1 - distance) }
func (s *Stack) At(index int) (Value, error) {
	if index < 0 || index >= len(s.values) { return Nil(), fmt.Errorf("stack index %d is out of range", index) }
	return s.values[index], nil
}
func (s *Stack) Set(index int, value Value) error {
	if index < 0 || index >= len(s.values) { return fmt.Errorf("stack index %d is out of range", index) }
	s.values[index] = value
	return nil
}
func (s *Stack) Truncate(size int) error {
	if size < 0 || size > len(s.values) { return fmt.Errorf("cannot truncate stack to %d", size) }
	for i := size; i < len(s.values); i++ {
		s.values[i] = Nil()
	}
	s.values = s.values[:size]
	return nil
}
func (s *Stack) Len() int  { return len(s.values) }
func (s *Stack) Peak() int { return s.peak }
func (s *Stack) Snapshot() []Value {
	result := make([]Value, len(s.values))
	copy(result, s.values)
	return result
}
func (s *Stack) Reset() {
	for i := range s.values {
		s.values[i] = Nil()
	}
	s.values = s.values[:0]
	s.peak = 0
}
type Frame struct {
	Closure *Closure
	IP      int
	Base    int
}
func (f Frame) Name() string {
	if f.Closure == nil || f.Closure.Function == nil { return "<invalid>" }
	return f.Closure.Function.DisplayName()
}
