package compiler
import (
	"fmt"
	"stackbyte-vm/internal/bytecode"
	"stackbyte-vm/internal/lexer"
	vmruntime "stackbyte-vm/internal/runtime"
)
type precedence uint8
const (
	precNone precedence = iota
	precAssignment
	precOr
	precAnd
	precEquality
	precComparison
	precTerm
	precFactor
	precUnary
	precCall
	precPrimary
)
type local struct {
	name       string
	depth      int
	isCaptured bool
}
type functionCompiler struct {
	enclosing  *functionCompiler
	function   *vmruntime.Function
	locals     []local
	scopeDepth int
	isScript   bool
}
type Compiler struct {
	parser  *parser
	current *functionCompiler
}
func Compile(source string) (*vmruntime.Function, error) {
	if err := lexer.ValidUTF8(source); err != nil { return nil, err }
	tokens, scanErrors := lexer.Scan(source)
	p := newParser(tokens, scanErrors)
	script := newFunctionCompiler(nil, "", true)
	c := &Compiler{parser: p, current: script}
	for !p.check(lexer.EOF) {
		c.declaration()
	}
	c.emitReturn()
	if p.failed() { return nil, &CompileError{Diagnostics: p.diagnostics} }
	if err := script.function.Validate(); err != nil { return nil, fmt.Errorf("validate compiled bytecode: %w", err) }
	return script.function, nil
}
func newFunctionCompiler(enclosing *functionCompiler, name string, script bool) *functionCompiler {
	fn := vmruntime.NewFunction(name)
	fc := &functionCompiler{enclosing: enclosing, function: fn, isScript: script}
	fc.locals = append(fc.locals, local{name: "", depth: 0})
	return fc
}
func (c *Compiler) declaration() {
	if c.parser.match(lexer.Fun) { c.funDeclaration() } else if c.parser.match(lexer.Var) { c.varDeclaration() } else { c.statement() }
	if c.parser.panicMode { c.parser.synchronize() }
}
func (c *Compiler) funDeclaration() {
	global := c.parseVariable("expected function name")
	name := c.parser.previous.Lexeme
	c.markInitialized()
	c.compileFunction(name)
	c.defineVariable(global)
}
func (c *Compiler) compileFunction(name string) {
	parent := c.current
	child := newFunctionCompiler(parent, name, false)
	c.current = child
	c.beginScope()
	c.parser.consume(lexer.LeftParen, "expected '(' after function name")
	if !c.parser.check(lexer.RightParen) {
		for {
			child.function.Arity++
			if child.function.Arity > 255 { c.parser.errorPrevious("function cannot have more than 255 parameters") }
			parameter := c.parseVariable("expected parameter name")
			c.defineVariable(parameter)
			if !c.parser.match(lexer.Comma) { break }
		}
	}
	c.parser.consume(lexer.RightParen, "expected ')' after parameters")
	c.parser.consume(lexer.LeftBrace, "expected '{' before function body")
	c.block()
	c.emitReturn()
	function := child.function
	c.current = parent
	constant := c.makeConstant(function)
	c.emitBytes(byte(bytecode.OpClosure), constant)
}
func (c *Compiler) varDeclaration() {
	global := c.parseVariable("expected variable name")
	if c.parser.match(lexer.Equal) { c.expression() } else { c.emitByte(byte(bytecode.OpNil)) }
	c.parser.consume(lexer.Semicolon, "expected ';' after variable declaration")
	c.defineVariable(global)
}
func (c *Compiler) statement() {
	switch {
	case c.parser.match(lexer.Print): c.printStatement()
	case c.parser.match(lexer.If): c.ifStatement()
	case c.parser.match(lexer.While): c.whileStatement()
	case c.parser.match(lexer.Return): c.returnStatement()
	case c.parser.match(lexer.LeftBrace): c.beginScope()
		c.block()
		c.endScope()
	default: c.expressionStatement()
	}
}
func (c *Compiler) printStatement() {
	c.expression()
	c.parser.consume(lexer.Semicolon, "expected ';' after value")
	c.emitByte(byte(bytecode.OpPrint))
}
func (c *Compiler) expressionStatement() {
	c.expression()
	c.parser.consume(lexer.Semicolon, "expected ';' after expression")
	c.emitByte(byte(bytecode.OpPop))
}
func (c *Compiler) returnStatement() {
	if c.current.isScript { c.parser.errorPrevious("cannot return from top-level code") }
	if c.parser.match(lexer.Semicolon) { c.emitReturn() } else {
		c.expression()
		c.parser.consume(lexer.Semicolon, "expected ';' after return value")
		c.emitByte(byte(bytecode.OpReturn))
	}
}
func (c *Compiler) ifStatement() {
	c.parser.consume(lexer.LeftParen, "expected '(' after if")
	c.expression()
	c.parser.consume(lexer.RightParen, "expected ')' after condition")
	thenJump := c.emitJump(bytecode.OpJumpIfFalse)
	c.emitByte(byte(bytecode.OpPop))
	c.statement()
	elseJump := c.emitJump(bytecode.OpJump)
	c.patchJump(thenJump)
	c.emitByte(byte(bytecode.OpPop))
	if c.parser.match(lexer.Else) { c.statement() }
	c.patchJump(elseJump)
}
func (c *Compiler) whileStatement() {
	loopStart := len(c.chunk().Code)
	c.parser.consume(lexer.LeftParen, "expected '(' after while")
	c.expression()
	c.parser.consume(lexer.RightParen, "expected ')' after condition")
	exitJump := c.emitJump(bytecode.OpJumpIfFalse)
	c.emitByte(byte(bytecode.OpPop))
	c.statement()
	if err := c.chunk().EmitLoop(loopStart, c.line()); err != nil { c.parser.errorPrevious(err.Error()) }
	c.patchJump(exitJump)
	c.emitByte(byte(bytecode.OpPop))
}
func (c *Compiler) block() {
	for !c.parser.check(lexer.RightBrace) && !c.parser.check(lexer.EOF) {
		c.declaration()
	}
	c.parser.consume(lexer.RightBrace, "expected '}' after block")
}
func (c *Compiler) expression() { c.parsePrecedence(precAssignment) }
func (c *Compiler) parsePrecedence(level precedence) {
	c.parser.advance()
	canAssign := level <= precAssignment
	switch c.parser.previous.Type {
	case lexer.Number: c.emitConstant(vmruntime.Number(c.parser.previous.Value.(float64)))
	case lexer.String: c.emitConstant(vmruntime.String(c.parser.previous.Value.(string)))
	case lexer.False: c.emitByte(byte(bytecode.OpFalse))
	case lexer.True: c.emitByte(byte(bytecode.OpTrue))
	case lexer.Nil: c.emitByte(byte(bytecode.OpNil))
	case lexer.LeftParen: c.expression()
		c.parser.consume(lexer.RightParen, "expected ')' after expression")
	case lexer.Minus, lexer.Bang: c.unary(c.parser.previous.Type)
	case lexer.Identifier: c.namedVariable(c.parser.previous, canAssign)
	default: c.parser.errorPrevious("expected expression")
		return
	}
	for level <= c.precedenceOf(c.parser.currentToken().Type) {
		c.parser.advance()
		operator := c.parser.previous.Type
		if operator == lexer.LeftParen { c.call() } else if operator == lexer.And { c.and() } else if operator == lexer.Or { c.or() } else { c.binary(operator) }
	}
	if canAssign && c.parser.match(lexer.Equal) { c.parser.errorPrevious("invalid assignment target") }
}
func (c *Compiler) unary(operator lexer.TokenType) {
	c.parsePrecedence(precUnary)
	if operator == lexer.Bang { c.emitByte(byte(bytecode.OpNot)) } else { c.emitByte(byte(bytecode.OpNegate)) }
}
func (c *Compiler) binary(operator lexer.TokenType) {
	precedence := c.precedenceOf(operator)
	c.parsePrecedence(precedence + 1)
	operations := map[lexer.TokenType][]byte{
		lexer.BangEqual:    {byte(bytecode.OpEqual), byte(bytecode.OpNot)},
		lexer.EqualEqual:   {byte(bytecode.OpEqual)},
		lexer.Greater:      {byte(bytecode.OpGreater)},
		lexer.GreaterEqual: {byte(bytecode.OpLess), byte(bytecode.OpNot)},
		lexer.Less:         {byte(bytecode.OpLess)},
		lexer.LessEqual:    {byte(bytecode.OpGreater), byte(bytecode.OpNot)},
		lexer.Plus:         {byte(bytecode.OpAdd)},
		lexer.Minus:        {byte(bytecode.OpSubtract)},
		lexer.Star:         {byte(bytecode.OpMultiply)},
		lexer.Slash:        {byte(bytecode.OpDivide)},
	}
	for _, op := range operations[operator] {
		c.emitByte(op)
	}
}
func (c *Compiler) and() {
	endJump := c.emitJump(bytecode.OpJumpIfFalse)
	c.emitByte(byte(bytecode.OpPop))
	c.parsePrecedence(precAnd)
	c.patchJump(endJump)
}
func (c *Compiler) or() {
	elseJump := c.emitJump(bytecode.OpJumpIfFalse)
	endJump := c.emitJump(bytecode.OpJump)
	c.patchJump(elseJump)
	c.emitByte(byte(bytecode.OpPop))
	c.parsePrecedence(precOr)
	c.patchJump(endJump)
}
func (c *Compiler) call() {
	count := 0
	if !c.parser.check(lexer.RightParen) {
		for {
			c.expression()
			count++
			if count > 255 { c.parser.errorPrevious("cannot have more than 255 arguments") }
			if !c.parser.match(lexer.Comma) { break }
		}
	}
	c.parser.consume(lexer.RightParen, "expected ')' after arguments")
	c.emitBytes(byte(bytecode.OpCall), byte(count))
}
func (c *Compiler) namedVariable(name lexer.Token, canAssign bool) {
	getOp, setOp := bytecode.OpGetGlobal, bytecode.OpSetGlobal
	argument := byte(0)
	if localIndex := c.resolveLocal(c.current, name); localIndex != -1 { getOp, setOp, argument = bytecode.OpGetLocal, bytecode.OpSetLocal, byte(localIndex) } else if upvalue := c.resolveUpvalue(c.current, name); upvalue != -1 { getOp, setOp, argument = bytecode.OpGetUpvalue, bytecode.OpSetUpvalue, byte(upvalue) } else { argument = c.identifierConstant(name) }
	if canAssign && c.parser.match(lexer.Equal) {
		c.expression()
		c.emitBytes(byte(setOp), argument)
	} else {
		c.emitBytes(byte(getOp), argument)
	}
}
func (c *Compiler) precedenceOf(kind lexer.TokenType) precedence {
	switch kind {
	case lexer.Or: return precOr
	case lexer.And: return precAnd
	case lexer.BangEqual, lexer.EqualEqual: return precEquality
	case lexer.Greater, lexer.GreaterEqual, lexer.Less, lexer.LessEqual: return precComparison
	case lexer.Plus, lexer.Minus: return precTerm
	case lexer.Star, lexer.Slash: return precFactor
	case lexer.LeftParen: return precCall
	default: return precNone
	}
}
func (c *Compiler) parseVariable(message string) byte {
	token := c.parser.consume(lexer.Identifier, message)
	c.declareVariable(token)
	if c.current.scopeDepth > 0 { return 0 }
	return c.identifierConstant(token)
}
func (c *Compiler) identifierConstant(token lexer.Token) byte { return c.makeConstant(vmruntime.String(token.Lexeme)) }
func (c *Compiler) declareVariable(name lexer.Token) {
	if c.current.scopeDepth == 0 { return }
	for i := len(c.current.locals) - 1; i >= 0; i-- {
		item := c.current.locals[i]
		if item.depth != -1 && item.depth < c.current.scopeDepth { break }
		if item.name == name.Lexeme { c.parser.errorAt(name, "variable already declared in this scope") }
	}
	if len(c.current.locals) >= 256 {
		c.parser.errorAt(name, "too many local variables in function")
		return
	}
	c.current.locals = append(c.current.locals, local{name: name.Lexeme, depth: -1})
}
func (c *Compiler) defineVariable(global byte) {
	if c.current.scopeDepth > 0 {
		c.markInitialized()
		return
	}
	c.emitBytes(byte(bytecode.OpDefineGlobal), global)
}
func (c *Compiler) markInitialized() {
	if c.current.scopeDepth == 0 || len(c.current.locals) == 0 { return }
	c.current.locals[len(c.current.locals)-1].depth = c.current.scopeDepth
}
func (c *Compiler) resolveLocal(fc *functionCompiler, name lexer.Token) int {
	for i := len(fc.locals) - 1; i >= 0; i-- {
		if fc.locals[i].name == name.Lexeme {
			if fc.locals[i].depth == -1 { c.parser.errorAt(name, "cannot read local variable in its own initializer") }
			return i
		}
	}
	return -1
}
func (c *Compiler) resolveUpvalue(fc *functionCompiler, name lexer.Token) int {
	if fc.enclosing == nil { return -1 }
	if localIndex := c.resolveLocal(fc.enclosing, name); localIndex != -1 {
		fc.enclosing.locals[localIndex].isCaptured = true
		index, err := fc.function.AddUpvalue(byte(localIndex), true)
		if err != nil { c.parser.errorAt(name, err.Error()) }
		return int(index)
	}
	if parentUpvalue := c.resolveUpvalue(fc.enclosing, name); parentUpvalue != -1 {
		index, err := fc.function.AddUpvalue(byte(parentUpvalue), false)
		if err != nil { c.parser.errorAt(name, err.Error()) }
		return int(index)
	}
	return -1
}
func (c *Compiler) beginScope() { c.current.scopeDepth++ }
func (c *Compiler) endScope() {
	c.current.scopeDepth--
	for len(c.current.locals) > 0 && c.current.locals[len(c.current.locals)-1].depth > c.current.scopeDepth {
		last := c.current.locals[len(c.current.locals)-1]
		if last.isCaptured { c.emitByte(byte(bytecode.OpCloseUpvalue)) } else { c.emitByte(byte(bytecode.OpPop)) }
		c.current.locals = c.current.locals[:len(c.current.locals)-1]
	}
}
func (c *Compiler) emitReturn() {
	c.emitByte(byte(bytecode.OpNil))
	c.emitByte(byte(bytecode.OpReturn))
}
func (c *Compiler) emitConstant(value vmruntime.Value) {
	constant := c.makeConstant(value)
	c.emitBytes(byte(bytecode.OpConstant), constant)
}
func (c *Compiler) makeConstant(value any) byte {
	index, err := c.chunk().AddConstant(value)
	if err != nil {
		c.parser.errorPrevious(err.Error())
		return 0
	}
	return index
}
func (c *Compiler) emitByte(value byte) { c.chunk().Write(value, c.line()) }
func (c *Compiler) emitBytes(first, second byte) {
	c.emitByte(first)
	c.emitByte(second)
}
func (c *Compiler) emitJump(op bytecode.OpCode) int { return c.chunk().EmitJump(op, c.line()) }
func (c *Compiler) patchJump(offset int) {
	if err := c.chunk().PatchJump(offset); err != nil { c.parser.errorPrevious(err.Error()) }
}
func (c *Compiler) line() int {
	if c.parser.previous.Line == 0 { return 1 }
	return c.parser.previous.Line
}
func (c *Compiler) chunk() *bytecode.Chunk { return c.current.function.Chunk }
