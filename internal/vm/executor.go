package vm
import (
	"context"
	"stackbyte-vm/internal/bytecode"
	vmruntime "stackbyte-vm/internal/runtime"
)
func (v *VM) Step(ctx context.Context) error {
	if v.done { return nil }
	select {
	case <-ctx.Done(): return v.runtimeError("execution canceled: %v", ctx.Err())
	default:
	}
	if v.stats.Instructions >= v.limits.MaxSteps { return v.runtimeError("instruction limit %d exceeded", v.limits.MaxSteps) }
	frame, chunk, op, err := v.readOp()
	if err != nil { return err }
	v.stats.Instructions++
	switch op {
	case bytecode.OpConstant: constant, err := v.readConstant(frame, chunk)
		if err != nil { return err }
		return v.pushConstant(constant)
	case bytecode.OpNil: return v.stack.Push(vmruntime.Nil())
	case bytecode.OpTrue: return v.stack.Push(vmruntime.Bool(true))
	case bytecode.OpFalse: return v.stack.Push(vmruntime.Bool(false))
	case bytecode.OpPop: _, err := v.stack.Pop()
		return err
	case bytecode.OpDup: value, err := v.stack.Peek(0)
		if err != nil { return err }
		return v.stack.Push(value)
	case bytecode.OpGetLocal: slot, err := v.readByte(frame, chunk)
		if err != nil { return err }
		value, err := v.stack.At(frame.Base + int(slot))
		if err != nil { return err }
		return v.stack.Push(value)
	case bytecode.OpSetLocal: slot, err := v.readByte(frame, chunk)
		if err != nil { return err }
		value, err := v.stack.Peek(0)
		if err != nil { return err }
		return v.stack.Set(frame.Base+int(slot), value)
	case bytecode.OpGetGlobal: name, err := v.readName(frame, chunk)
		if err != nil { return err }
		value, ok := v.globals[name]
		if !ok { return v.runtimeError("undefined variable %q", name) }
		return v.stack.Push(value)
	case bytecode.OpDefineGlobal: name, err := v.readName(frame, chunk)
		if err != nil { return err }
		if _, exists := v.globals[name]; exists { return v.runtimeError("global %q is already defined", name) }
		value, err := v.stack.Pop()
		if err != nil { return err }
		v.globals[name] = value
		return nil
	case bytecode.OpSetGlobal: name, err := v.readName(frame, chunk)
		if err != nil { return err }
		if _, exists := v.globals[name]; !exists { return v.runtimeError("undefined variable %q", name) }
		value, err := v.stack.Peek(0)
		if err != nil { return err }
		v.globals[name] = value
		return nil
	case bytecode.OpGetUpvalue: return v.getUpvalue(frame, chunk)
	case bytecode.OpSetUpvalue: return v.setUpvalue(frame, chunk)
	case bytecode.OpEqual: b, err := v.stack.Pop()
		if err != nil { return err }
		a, err := v.stack.Pop()
		if err != nil { return err }
		return v.stack.Push(vmruntime.Bool(a.Equal(b)))
	case bytecode.OpGreater: return v.numericCompare(func(a, b float64) bool { return a > b })
	case bytecode.OpLess: return v.numericCompare(func(a, b float64) bool { return a < b })
	case bytecode.OpAdd: return v.add()
	case bytecode.OpSubtract: return v.numericBinary("-", func(a, b float64) float64 { return a - b })
	case bytecode.OpMultiply: return v.numericBinary("*", func(a, b float64) float64 { return a * b })
	case bytecode.OpDivide: return v.divide()
	case bytecode.OpNot: value, err := v.stack.Pop()
		if err != nil { return err }
		return v.stack.Push(vmruntime.Bool(value.IsFalsey()))
	case bytecode.OpNegate: value, err := v.stack.Pop()
		if err != nil { return err }
		n, ok := value.Number()
		if !ok { return v.runtimeError("unary '-' requires number") }
		return v.stack.Push(vmruntime.Number(-n))
	case bytecode.OpPrint: return v.printValue()
	case bytecode.OpJump: offset, err := v.readShort(frame, chunk)
		if err != nil { return err }
		frame.IP += offset
		return nil
	case bytecode.OpJumpIfFalse: offset, err := v.readShort(frame, chunk)
		if err != nil { return err }
		value, err := v.stack.Peek(0)
		if err != nil { return err }
		if value.IsFalsey() { frame.IP += offset }
		return nil
	case bytecode.OpLoop: offset, err := v.readShort(frame, chunk)
		if err != nil { return err }
		frame.IP -= offset
		return nil
	case bytecode.OpCall: count, err := v.readByte(frame, chunk)
		if err != nil { return err }
		return v.callValue(int(count))
	case bytecode.OpClosure: return v.createClosure(frame, chunk)
	case bytecode.OpCloseUpvalue: v.closeUpvalues(v.stack.Len() - 1)
		_, err := v.stack.Pop()
		return err
	case bytecode.OpReturn: return v.returnFromCall()
	default: return v.runtimeError("unknown opcode %d", op)
	}
}
func (v *VM) readOp() (*vmruntime.Frame, *bytecode.Chunk, bytecode.OpCode, error) {
	if len(v.frames) == 0 { return nil, nil, 0, v.runtimeError("no active call frame") }
	frame := &v.frames[len(v.frames)-1]
	chunk := frame.Closure.Function.Chunk
	if frame.IP < 0 || frame.IP >= len(chunk.Code) { return nil, nil, 0, v.runtimeError("instruction pointer %d is out of range", frame.IP) }
	op := bytecode.OpCode(chunk.Code[frame.IP])
	frame.IP++
	return frame, chunk, op, nil
}
func (v *VM) readByte(frame *vmruntime.Frame, chunk *bytecode.Chunk) (byte, error) {
	if frame.IP >= len(chunk.Code) { return 0, v.runtimeError("truncated bytecode operand") }
	value := chunk.Code[frame.IP]
	frame.IP++
	return value, nil
}
func (v *VM) readShort(frame *vmruntime.Frame, chunk *bytecode.Chunk) (int, error) {
	high, err := v.readByte(frame, chunk)
	if err != nil { return 0, err }
	low, err := v.readByte(frame, chunk)
	if err != nil { return 0, err }
	return bytecode.DecodeUint16(high, low), nil
}
func (v *VM) readConstant(frame *vmruntime.Frame, chunk *bytecode.Chunk) (any, error) {
	index, err := v.readByte(frame, chunk)
	if err != nil { return nil, err }
	if int(index) >= len(chunk.Constants) { return nil, v.runtimeError("constant index %d is out of range", index) }
	return chunk.Constants[index], nil
}
func (v *VM) readName(frame *vmruntime.Frame, chunk *bytecode.Chunk) (string, error) {
	constant, err := v.readConstant(frame, chunk)
	if err != nil { return "", err }
	value, ok := constant.(vmruntime.Value)
	if !ok { return "", v.runtimeError("global name constant has invalid type") }
	name, ok := value.StringValue()
	if !ok { return "", v.runtimeError("global name must be a string") }
	return name, nil
}
func (v *VM) pushConstant(constant any) error {
	value, ok := constant.(vmruntime.Value)
	if !ok { return v.runtimeError("constant is not a runtime value") }
	return v.stack.Push(value)
}
func (v *VM) numericOperands() (float64, float64, error) {
	bv, err := v.stack.Pop()
	if err != nil { return 0, 0, err }
	av, err := v.stack.Pop()
	if err != nil { return 0, 0, err }
	a, aok := av.Number()
	b, bok := bv.Number()
	if !aok || !bok { return 0, 0, v.runtimeError("operands must be numbers") }
	return a, b, nil
}
func (v *VM) numericBinary(name string, operation func(float64, float64) float64) error {
	a, b, err := v.numericOperands()
	if err != nil { return err }
	result := operation(a, b)
	if name == "/" && b == 0 { return v.runtimeError("division by zero") }
	return v.stack.Push(vmruntime.Number(result))
}
func (v *VM) numericCompare(operation func(float64, float64) bool) error {
	a, b, err := v.numericOperands()
	if err != nil { return err }
	return v.stack.Push(vmruntime.Bool(operation(a, b)))
}
func (v *VM) add() error {
	b, err := v.stack.Pop()
	if err != nil { return err }
	a, err := v.stack.Pop()
	if err != nil { return err }
	if an, ok := a.Number(); ok {
		if bn, ok := b.Number(); ok { return v.stack.Push(vmruntime.Number(an + bn)) }
	}
	if as, ok := a.StringValue(); ok {
		if bs, ok := b.StringValue(); ok { return v.stack.Push(vmruntime.String(as + bs)) }
	}
	return v.runtimeError("'+' requires two numbers or two strings")
}
func (v *VM) divide() error {
	a, b, err := v.numericOperands()
	if err != nil { return err }
	if b == 0 { return v.runtimeError("division by zero") }
	return v.stack.Push(vmruntime.Number(a / b))
}
func (v *VM) printValue() error {
	value, err := v.stack.Pop()
	if err != nil { return err }
	text := value.String() + "\n"
	if v.output.Len()+len(text) > v.limits.MaxOutputBytes { return v.runtimeError("output limit %d exceeded", v.limits.MaxOutputBytes) }
	_, _ = v.output.WriteString(text)
	return nil
}
func (v *VM) getUpvalue(frame *vmruntime.Frame, chunk *bytecode.Chunk) error {
	index, err := v.readByte(frame, chunk)
	if err != nil { return err }
	upvalue, err := frame.Closure.GetUpvalue(int(index))
	if err != nil { return v.runtimeError("%v", err) }
	value, err := upvalue.Get()
	if err != nil { return err }
	return v.stack.Push(value)
}
func (v *VM) setUpvalue(frame *vmruntime.Frame, chunk *bytecode.Chunk) error {
	index, err := v.readByte(frame, chunk)
	if err != nil { return err }
	upvalue, err := frame.Closure.GetUpvalue(int(index))
	if err != nil { return v.runtimeError("%v", err) }
	value, err := v.stack.Peek(0)
	if err != nil { return err }
	return upvalue.Set(value)
}
func (v *VM) createClosure(frame *vmruntime.Frame, chunk *bytecode.Chunk) error {
	constant, err := v.readConstant(frame, chunk)
	if err != nil { return err }
	function, ok := constant.(*vmruntime.Function)
	if !ok { return v.runtimeError("CLOSURE constant is not a function") }
	closure := vmruntime.NewClosure(function)
	for i, spec := range function.Upvalues {
		var upvalue *vmruntime.Upvalue
		if spec.IsLocal { upvalue = v.captureUpvalue(frame.Base + int(spec.Index)) } else { upvalue, err = frame.Closure.GetUpvalue(int(spec.Index)) }
		if err != nil { return err }
		if err := closure.SetUpvalue(i, upvalue); err != nil { return err }
	}
	return v.stack.Push(vmruntime.ClosureValue(closure))
}
func (v *VM) captureUpvalue(index int) *vmruntime.Upvalue {
	if existing, ok := v.open[index]; ok { return existing }
	upvalue := vmruntime.NewOpenUpvalue(v.stack, index)
	v.open[index] = upvalue
	return upvalue
}
func (v *VM) closeUpvalues(last int) {
	for index, upvalue := range v.open {
		if index >= last {
			_ = upvalue.Close()
			delete(v.open, index)
		}
	}
}
func (v *VM) callValue(argCount int) error {
	callee, err := v.stack.Peek(argCount)
	if err != nil { return err }
	if closure, ok := callee.Closure(); ok { return v.callClosure(closure, argCount) }
	if native, ok := callee.Native(); ok {
		base := v.stack.Len() - argCount - 1
		values := v.stack.Snapshot()
		result, err := native.Invoke(values[base+1:])
		if err != nil { return v.runtimeError("%v", err) }
		if err := v.stack.Truncate(base); err != nil { return err }
		return v.stack.Push(result)
	}
	return v.runtimeError("value of type %s is not callable", callee.TypeName())
}
func (v *VM) callClosure(closure *vmruntime.Closure, argCount int) error {
	if closure.Function.Arity != argCount { return v.runtimeError("%s expects %d arguments, got %d", closure.Function.DisplayName(), closure.Function.Arity, argCount) }
	if len(v.frames) >= v.limits.MaxFrames { return v.runtimeError("call frame limit %d exceeded", v.limits.MaxFrames) }
	base := v.stack.Len() - argCount - 1
	v.frames = append(v.frames, vmruntime.Frame{Closure: closure, Base: base})
	if len(v.frames) > v.stats.MaxFrames { v.stats.MaxFrames = len(v.frames) }
	return nil
}
func (v *VM) returnFromCall() error {
	result, err := v.stack.Pop()
	if err != nil { return err }
	frame := v.frames[len(v.frames)-1]
	v.closeUpvalues(frame.Base)
	v.frames = v.frames[:len(v.frames)-1]
	if err := v.stack.Truncate(frame.Base); err != nil { return err }
	if len(v.frames) == 0 {
		v.done = true
		v.result = result
		return nil
	}
	return v.stack.Push(result)
}
