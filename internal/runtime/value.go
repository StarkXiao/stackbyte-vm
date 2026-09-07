package runtime
import (
	"fmt"
	"math"
	"strconv"
)
type ValueType uint8
const (
	NilType ValueType = iota
	NumberType
	BoolType
	StringType
	FunctionType
	ClosureType
	NativeType
)
type Value struct {
	kind ValueType
	data any
}
func Nil() Value                      { return Value{kind: NilType} }
func Number(v float64) Value          { return Value{kind: NumberType, data: v} }
func Bool(v bool) Value               { return Value{kind: BoolType, data: v} }
func String(v string) Value           { return Value{kind: StringType, data: v} }
func FunctionValue(v *Function) Value { return Value{kind: FunctionType, data: v} }
func ClosureValue(v *Closure) Value   { return Value{kind: ClosureType, data: v} }
func NativeValue(v *Native) Value     { return Value{kind: NativeType, data: v} }
func (v Value) Type() ValueType       { return v.kind }
func (v Value) IsNil() bool           { return v.kind == NilType }
func (v Value) TypeName() string {
	switch v.kind {
	case NilType: return "nil"
	case NumberType: return "number"
	case BoolType: return "bool"
	case StringType: return "string"
	case FunctionType: return "function"
	case ClosureType: return "closure"
	case NativeType: return "native"
	default: return "unknown"
	}
}
func (v Value) Number() (float64, bool) {
	value, ok := v.data.(float64)
	return value, ok && v.kind == NumberType
}
func (v Value) Bool() (bool, bool) {
	value, ok := v.data.(bool)
	return value, ok && v.kind == BoolType
}
func (v Value) StringValue() (string, bool) {
	value, ok := v.data.(string)
	return value, ok && v.kind == StringType
}
func (v Value) Function() (*Function, bool) {
	value, ok := v.data.(*Function)
	return value, ok && v.kind == FunctionType
}
func (v Value) Closure() (*Closure, bool) {
	value, ok := v.data.(*Closure)
	return value, ok && v.kind == ClosureType
}
func (v Value) Native() (*Native, bool) {
	value, ok := v.data.(*Native)
	return value, ok && v.kind == NativeType
}
func (v Value) IsFalsey() bool {
	if v.kind == NilType { return true }
	value, ok := v.Bool()
	return ok && !value
}
func (v Value) Equal(other Value) bool {
	if v.kind != other.kind { return false }
	switch v.kind {
	case NilType: return true
	case NumberType: a, _ := v.Number()
		b, _ := other.Number()
		return a == b || math.IsNaN(a) && math.IsNaN(b)
	case BoolType: a, _ := v.Bool()
		b, _ := other.Bool()
		return a == b
	case StringType: a, _ := v.StringValue()
		b, _ := other.StringValue()
		return a == b
	default: return v.data == other.data
	}
}
func (v Value) String() string {
	switch v.kind {
	case NilType: return "nil"
	case NumberType: n, _ := v.Number()
		return strconv.FormatFloat(n, 'g', -1, 64)
	case BoolType: b, _ := v.Bool()
		return strconv.FormatBool(b)
	case StringType: s, _ := v.StringValue()
		return s
	case FunctionType: fn, _ := v.Function()
		return fmt.Sprintf("<fn %s>", fn.DisplayName())
	case ClosureType: closure, _ := v.Closure()
		return fmt.Sprintf("<fn %s>", closure.Function.DisplayName())
	case NativeType: native, _ := v.Native()
		return fmt.Sprintf("<native %s>", native.Name)
	default: return "<invalid>"
	}
}
