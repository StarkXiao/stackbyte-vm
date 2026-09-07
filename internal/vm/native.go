package vm
import (
	"fmt"
	vmruntime "stackbyte-vm/internal/runtime"
	"time"
	"unicode/utf8"
)
func (v *VM) installNatives() {
	v.defineNative("clock", 0, 0, func(_ []vmruntime.Value) (vmruntime.Value, error) {
		return vmruntime.Number(float64(time.Now().UnixNano()) / 1e9), nil
	})
	v.defineNative("type", 1, 1, func(args []vmruntime.Value) (vmruntime.Value, error) {
		return vmruntime.String(args[0].TypeName()), nil
	})
	v.defineNative("len", 1, 1, func(args []vmruntime.Value) (vmruntime.Value, error) {
		value, ok := args[0].StringValue()
		if !ok { return vmruntime.Nil(), fmt.Errorf("len expects a string") }
		return vmruntime.Number(float64(utf8.RuneCountInString(value))), nil
	})
}
func (v *VM) defineNative(name string, minArity, maxArity int, call vmruntime.NativeFunc) {
	v.globals[name] = vmruntime.NativeValue(&vmruntime.Native{
		Name: name, MinArity: minArity, MaxArity: maxArity, Call: call,
	})
}
func (v *VM) GlobalsSnapshot() map[string]string {
	result := make(map[string]string, len(v.globals))
	for name, value := range v.globals {
		result[name] = value.String()
	}
	return result
}
