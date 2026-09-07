package compiler
import (
	"fmt"
	"stackbyte-vm/internal/lexer"
)
type Diagnostic struct {
	Phase   string `json:"phase"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Message string `json:"message"`
}
type CompileError struct {
	Diagnostics []Diagnostic `json:"diagnostics"`
}
func (e *CompileError) Error() string {
	if len(e.Diagnostics) == 0 { return "compilation failed" }
	first := e.Diagnostics[0]
	return fmt.Sprintf("%s error at %d:%d: %s", first.Phase, first.Line, first.Column, first.Message)
}
type parser struct {
	tokens      []lexer.Token
	current     int
	previous    lexer.Token
	diagnostics []Diagnostic
	panicMode   bool
}
func newParser(tokens []lexer.Token, scanErrors []lexer.Diagnostic) *parser {
	p := &parser{tokens: tokens}
	for _, item := range scanErrors {
		p.diagnostics = append(p.diagnostics, Diagnostic{
			Phase: "scan", Line: item.Line, Column: item.Column, Message: item.Message,
		})
	}
	return p
}
func (p *parser) advance() lexer.Token {
	if p.current < len(p.tokens) {
		p.previous = p.tokens[p.current]
		p.current++
	}
	return p.previous
}
func (p *parser) currentToken() lexer.Token {
	if p.current >= len(p.tokens) { return lexer.Token{Type: lexer.EOF, Line: p.previous.Line, Column: p.previous.Column} }
	return p.tokens[p.current]
}
func (p *parser) check(kind lexer.TokenType) bool { return p.currentToken().Type == kind }
func (p *parser) match(kind lexer.TokenType) bool {
	if !p.check(kind) { return false }
	p.advance()
	return true
}
func (p *parser) consume(kind lexer.TokenType, message string) lexer.Token {
	if p.check(kind) { return p.advance() }
	p.errorAt(p.currentToken(), message)
	return p.currentToken()
}
func (p *parser) errorAt(token lexer.Token, message string) {
	if p.panicMode { return }
	p.panicMode = true
	if token.Type == lexer.EOF { message += " at end of input" }
	p.diagnostics = append(p.diagnostics, Diagnostic{
		Phase: "compile", Line: token.Line, Column: token.Column, Message: message,
	})
}
func (p *parser) errorPrevious(message string) {
	p.errorAt(p.previous, message)
}
func (p *parser) synchronize() {
	p.panicMode = false
	for !p.check(lexer.EOF) {
		if p.previous.Type == lexer.Semicolon { return }
		switch p.currentToken().Type {
		case lexer.Fun, lexer.Var, lexer.If, lexer.While, lexer.Print, lexer.Return: return
		default: p.advance()
		}
	}
}
func (p *parser) failed() bool { return len(p.diagnostics) > 0 }
