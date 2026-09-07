package bytecode
import (
	"fmt"
	"strings"
)
func Disassemble(name string, chunk *Chunk) string {
	var out strings.Builder
	fmt.Fprintf(&out, "== %s ==\n", name)
	for offset := 0; offset < len(chunk.Code); {
		next, text, err := DisassembleInstruction(chunk, offset)
		if err != nil {
			fmt.Fprintf(&out, "%04d ERROR %v\n", offset, err)
			break
		}
		out.WriteString(text)
		offset = next
	}
	return out.String()
}
func DisassembleInstruction(chunk *Chunk, offset int) (int, string, error) {
	next, err := instructionEnd(chunk, offset)
	if err != nil { return offset, "", err }
	op := OpCode(chunk.Code[offset])
	var out strings.Builder
	fmt.Fprintf(&out, "%04d %4d %-18s", offset, chunk.Line(offset), op.String())
	switch op {
	case OpConstant, OpGetGlobal, OpDefineGlobal, OpSetGlobal, OpClosure: index := int(chunk.Code[offset+1])
		fmt.Fprintf(&out, "%4d %v", index, chunk.Constants[index])
	case OpGetLocal, OpSetLocal, OpGetUpvalue, OpSetUpvalue, OpCall: fmt.Fprintf(&out, "%4d", chunk.Code[offset+1])
	case OpJump, OpJumpIfFalse: distance := DecodeUint16(chunk.Code[offset+1], chunk.Code[offset+2])
		fmt.Fprintf(&out, "%4d -> %d", distance, next+distance)
	case OpLoop: distance := DecodeUint16(chunk.Code[offset+1], chunk.Code[offset+2])
		fmt.Fprintf(&out, "%4d -> %d", distance, next-distance)
	}
	out.WriteByte('\n')
	return next, out.String(), nil
}
