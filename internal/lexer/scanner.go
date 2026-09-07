package lexer
import (
	"fmt"
	"strconv"
	"unicode/utf8"
)
type TokenType uint8
const (
	LeftParen TokenType = iota
	RightParen
	LeftBrace
	RightBrace
	Comma
	Minus
	Plus
	Semicolon
	Slash
	Star
	Bang
	BangEqual
	Equal
	EqualEqual
	Greater
	GreaterEqual
	Less
	LessEqual
	Identifier
	String
	Number
	And
	Else
	False
	Fun
	If
	Nil
	Or
	Print
	Return
	True
	Var
	While
	EOF
)
type Token struct {
	Type   TokenType
	Lexeme string
	Line   int
	Column int
	Value  any
}
type Diagnostic struct {
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Message string `json:"message"`
}
type Scanner struct {
	source      string
	start       int
	current     int
	line        int
	column      int
	startColumn int
	tokens      []Token
	errors      []Diagnostic
}
var keywords = map[string]TokenType{
	"and": And, "else": Else, "false": False, "fun": Fun,
	"if": If, "nil": Nil, "or": Or, "print": Print,
	"return": Return, "true": True, "var": Var, "while": While,
}
func Scan(source string) ([]Token, []Diagnostic) {
	s := &Scanner{source: source, line: 1, column: 1}
	for !s.atEnd() {
		s.start = s.current
		s.startColumn = s.column
		s.scanToken()
	}
	s.tokens = append(s.tokens, Token{Type: EOF, Line: s.line, Column: s.column})
	return s.tokens, s.errors
}
func (s *Scanner) scanToken() {
	c := s.advance()
	switch c {
	case '(': s.add(LeftParen, nil)
	case ')': s.add(RightParen, nil)
	case '{': s.add(LeftBrace, nil)
	case '}': s.add(RightBrace, nil)
	case ',': s.add(Comma, nil)
	case '-': s.add(Minus, nil)
	case '+': s.add(Plus, nil)
	case ';': s.add(Semicolon, nil)
	case '*': s.add(Star, nil)
	case '!': s.addIf('=', BangEqual, Bang)
	case '=': s.addIf('=', EqualEqual, Equal)
	case '<': s.addIf('=', LessEqual, Less)
	case '>': s.addIf('=', GreaterEqual, Greater)
	case '/': if s.match('/') {
			for s.peek() != '\n' && !s.atEnd() {
				s.advance()
			}
		} else {
			s.add(Slash, nil)
		}
	case ' ', '\r', '\t':
	case '\n': s.line++
		s.column = 1
	case '"': s.stringToken()
	default: if isDigit(c) { s.numberToken() } else if isAlpha(c) { s.identifier() } else { s.fail(fmt.Sprintf("unexpected character %q", c)) }
	}
}
func (s *Scanner) identifier() {
	for isAlphaNumeric(s.peek()) {
		s.advance()
	}
	text := s.source[s.start:s.current]
	typeName, ok := keywords[text]
	if !ok { typeName = Identifier }
	s.add(typeName, nil)
}
func (s *Scanner) numberToken() {
	for isDigit(s.peek()) {
		s.advance()
	}
	if s.peek() == '.' && isDigit(s.peekNext()) {
		s.advance()
		for isDigit(s.peek()) {
			s.advance()
		}
	}
	text := s.source[s.start:s.current]
	value, err := strconv.ParseFloat(text, 64)
	if err != nil {
		s.fail("invalid number literal")
		return
	}
	s.add(Number, value)
}
func (s *Scanner) stringToken() {
	for s.peek() != '"' && !s.atEnd() {
		if s.peek() == '\n' {
			s.line++
			s.column = 1
		}
		s.advance()
	}
	if s.atEnd() {
		s.fail("unterminated string")
		return
	}
	s.advance()
	raw := s.source[s.start+1 : s.current-1]
	s.add(String, raw)
}
func (s *Scanner) addIf(expected byte, yes, no TokenType) {
	if s.match(expected) { s.add(yes, nil) } else { s.add(no, nil) }
}
func (s *Scanner) add(kind TokenType, value any) {
	s.tokens = append(s.tokens, Token{
		Type: kind, Lexeme: s.source[s.start:s.current],
		Line: s.line, Column: s.startColumn, Value: value,
	})
}
func (s *Scanner) fail(message string) {
	s.errors = append(s.errors, Diagnostic{s.line, s.startColumn, message})
}
func (s *Scanner) atEnd() bool { return s.current >= len(s.source) }
func (s *Scanner) advance() byte {
	c := s.source[s.current]
	s.current++
	s.column++
	return c
}
func (s *Scanner) match(expected byte) bool {
	if s.atEnd() || s.source[s.current] != expected { return false }
	s.current++
	s.column++
	return true
}
func (s *Scanner) peek() byte {
	if s.atEnd() { return 0 }
	return s.source[s.current]
}
func (s *Scanner) peekNext() byte {
	if s.current+1 >= len(s.source) { return 0 }
	return s.source[s.current+1]
}
func isDigit(c byte) bool        { return c >= '0' && c <= '9' }
func isAlpha(c byte) bool        { return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }
func isAlphaNumeric(c byte) bool { return isAlpha(c) || isDigit(c) }
func ValidUTF8(source string) error {
	if !utf8.ValidString(source) { return fmt.Errorf("source is not valid UTF-8") }
	return nil
}
