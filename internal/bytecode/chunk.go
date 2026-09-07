package bytecode
import "fmt"
type Chunk struct {
	Code      []byte
	Lines     []int
	Constants []any
}
func NewChunk() *Chunk {
	return &Chunk{
		Code:      make([]byte, 0, 128),
		Lines:     make([]int, 0, 128),
		Constants: make([]any, 0, 32),
	}
}
func (c *Chunk) Write(value byte, line int) {
	c.Code = append(c.Code, value)
	c.Lines = append(c.Lines, line)
}
func (c *Chunk) WriteOp(op OpCode, line int) {
	c.Write(byte(op), line)
}
func (c *Chunk) WriteShort(value int, line int) error {
	high, low, err := EncodeUint16(value)
	if err != nil { return err }
	c.Write(high, line)
	c.Write(low, line)
	return nil
}
func (c *Chunk) AddConstant(value any) (byte, error) {
	if len(c.Constants) >= 256 { return 0, fmt.Errorf("a chunk cannot contain more than 256 constants") }
	c.Constants = append(c.Constants, value)
	return byte(len(c.Constants) - 1), nil
}
func (c *Chunk) EmitConstant(value any, line int) error {
	index, err := c.AddConstant(value)
	if err != nil { return err }
	c.WriteOp(OpConstant, line)
	c.Write(index, line)
	return nil
}
func (c *Chunk) EmitJump(op OpCode, line int) int {
	c.WriteOp(op, line)
	c.Write(0xff, line)
	c.Write(0xff, line)
	return len(c.Code) - 2
}
func (c *Chunk) PatchJump(operandOffset int) error {
	distance := len(c.Code) - operandOffset - 2
	high, low, err := EncodeUint16(distance)
	if err != nil { return fmt.Errorf("jump is too large: %w", err) }
	if operandOffset < 0 || operandOffset+1 >= len(c.Code) { return fmt.Errorf("invalid patch offset %d", operandOffset) }
	c.Code[operandOffset] = high
	c.Code[operandOffset+1] = low
	return nil
}
func (c *Chunk) EmitLoop(loopStart, line int) error {
	c.WriteOp(OpLoop, line)
	distance := len(c.Code) - loopStart + 2
	return c.WriteShort(distance, line)
}
func (c *Chunk) Line(offset int) int {
	if offset < 0 || offset >= len(c.Lines) { return 0 }
	return c.Lines[offset]
}
func (c *Chunk) Reset() {
	c.Code = c.Code[:0]
	c.Lines = c.Lines[:0]
	c.Constants = c.Constants[:0]
}
