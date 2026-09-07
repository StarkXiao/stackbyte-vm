package bytecode
import "fmt"
type ValidationError struct {
	Offset  int
	Message string
}
func (e ValidationError) Error() string { return fmt.Sprintf("bytecode at offset %d: %s", e.Offset, e.Message) }
func Validate(chunk *Chunk) error {
	if chunk == nil { return ValidationError{Message: "nil chunk"} }
	if len(chunk.Code) != len(chunk.Lines) { return ValidationError{Message: "code and source line tables differ in length"} }
	boundaries := make(map[int]bool, len(chunk.Code))
	for offset := 0; offset < len(chunk.Code); {
		boundaries[offset] = true
		next, err := instructionEnd(chunk, offset)
		if err != nil { return err }
		offset = next
	}
	boundaries[len(chunk.Code)] = true
	for offset := 0; offset < len(chunk.Code); {
		op := OpCode(chunk.Code[offset])
		next, _ := instructionEnd(chunk, offset)
		if IsJump(op) {
			distance := DecodeUint16(chunk.Code[offset+1], chunk.Code[offset+2])
			target := next + distance
			if op == OpLoop { target = next - distance }
			if target < 0 || target > len(chunk.Code) || !boundaries[target] { return ValidationError{offset, fmt.Sprintf("jump target %d is not an instruction boundary", target)} }
		}
		offset = next
	}
	if len(chunk.Code) == 0 || OpCode(chunk.Code[len(chunk.Code)-1]) != OpReturn { return ValidationError{len(chunk.Code), "chunk must end with RETURN"} }
	return nil
}
func instructionEnd(chunk *Chunk, offset int) (int, error) {
	if offset < 0 || offset >= len(chunk.Code) { return 0, ValidationError{offset, "instruction offset is out of range"} }
	op := OpCode(chunk.Code[offset])
	info, ok := op.Info()
	if !ok { return 0, ValidationError{offset, fmt.Sprintf("unknown opcode %d", byte(op))} }
	end := offset + 1 + info.OperandBytes
	if end > len(chunk.Code) { return 0, ValidationError{offset, "truncated instruction operand"} }
	if usesConstant(op) {
		index := int(chunk.Code[offset+1])
		if index >= len(chunk.Constants) { return 0, ValidationError{offset, fmt.Sprintf("constant index %d is out of range", index)} }
	}
	return end, nil
}
func usesConstant(op OpCode) bool {
	switch op {
	case OpConstant, OpGetGlobal, OpDefineGlobal, OpSetGlobal, OpClosure: return true
	default: return false
	}
}
func InstructionEnd(chunk *Chunk, offset int) (int, error) { return instructionEnd(chunk, offset) }
