package bytecode
import "fmt"
type OpCode byte
const (
	OpConstant OpCode = iota
	OpNil
	OpTrue
	OpFalse
	OpPop
	OpDup
	OpGetLocal
	OpSetLocal
	OpGetGlobal
	OpDefineGlobal
	OpSetGlobal
	OpGetUpvalue
	OpSetUpvalue
	OpEqual
	OpGreater
	OpLess
	OpAdd
	OpSubtract
	OpMultiply
	OpDivide
	OpNot
	OpNegate
	OpPrint
	OpJump
	OpJumpIfFalse
	OpLoop
	OpCall
	OpClosure
	OpCloseUpvalue
	OpReturn
)
type Info struct {
	Name         string
	OperandBytes int
	StackEffect  int
}
var infos = map[OpCode]Info{
	OpConstant:     {"CONSTANT", 1, 1},
	OpNil:          {"NIL", 0, 1},
	OpTrue:         {"TRUE", 0, 1},
	OpFalse:        {"FALSE", 0, 1},
	OpPop:          {"POP", 0, -1},
	OpDup:          {"DUP", 0, 1},
	OpGetLocal:     {"GET_LOCAL", 1, 1},
	OpSetLocal:     {"SET_LOCAL", 1, 0},
	OpGetGlobal:    {"GET_GLOBAL", 1, 1},
	OpDefineGlobal: {"DEFINE_GLOBAL", 1, -1},
	OpSetGlobal:    {"SET_GLOBAL", 1, 0},
	OpGetUpvalue:   {"GET_UPVALUE", 1, 1},
	OpSetUpvalue:   {"SET_UPVALUE", 1, 0},
	OpEqual:        {"EQUAL", 0, -1},
	OpGreater:      {"GREATER", 0, -1},
	OpLess:         {"LESS", 0, -1},
	OpAdd:          {"ADD", 0, -1},
	OpSubtract:     {"SUBTRACT", 0, -1},
	OpMultiply:     {"MULTIPLY", 0, -1},
	OpDivide:       {"DIVIDE", 0, -1},
	OpNot:          {"NOT", 0, 0},
	OpNegate:       {"NEGATE", 0, 0},
	OpPrint:        {"PRINT", 0, -1},
	OpJump:         {"JUMP", 2, 0},
	OpJumpIfFalse:  {"JUMP_IF_FALSE", 2, 0},
	OpLoop:         {"LOOP", 2, 0},
	OpCall:         {"CALL", 1, 0},
	OpClosure:      {"CLOSURE", 1, 1},
	OpCloseUpvalue: {"CLOSE_UPVALUE", 0, -1},
	OpReturn:       {"RETURN", 0, -1},
}
func (op OpCode) Info() (Info, bool) {
	info, ok := infos[op]
	return info, ok
}
func (op OpCode) String() string {
	if info, ok := op.Info(); ok { return info.Name }
	return fmt.Sprintf("UNKNOWN_%d", byte(op))
}
func IsJump(op OpCode) bool { return op == OpJump || op == OpJumpIfFalse || op == OpLoop }
func EncodeUint16(value int) (byte, byte, error) {
	if value < 0 || value > 65535 { return 0, 0, fmt.Errorf("operand %d exceeds uint16 range", value) }
	return byte(value >> 8), byte(value), nil
}
func DecodeUint16(high, low byte) int { return int(high)<<8 | int(low) }
