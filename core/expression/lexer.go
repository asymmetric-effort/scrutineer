package expression

import (
	"fmt"
	"strings"
	"unicode"
)

// Lexer tokenises an expression string (the content inside ${fn:...}).
type Lexer struct {
	input []rune
	pos   int
}

// NewLexer creates a Lexer for the given input.
func NewLexer(input string) *Lexer {
	return &Lexer{input: []rune(input)}
}

// Tokenize returns all tokens from the input.
func (l *Lexer) Tokenize() ([]Token, error) {
	var tokens []Token
	for {
		l.skipWhitespace()
		if l.pos >= len(l.input) {
			break
		}
		ch := l.input[l.pos]

		switch {
		case ch == '(':
			tokens = append(tokens, Token{Type: TokenLParen, Value: "("})
			l.pos++
		case ch == ')':
			tokens = append(tokens, Token{Type: TokenRParen, Value: ")"})
			l.pos++
		case ch == ',':
			tokens = append(tokens, Token{Type: TokenComma, Value: ","})
			l.pos++
		case ch == '"' || ch == '\'':
			tok, err := l.readString(ch)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, tok)
		case ch == '-' || isDigit(ch):
			tok, err := l.readNumber()
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, tok)
		case isIdentStart(ch):
			tok := l.readIdent()
			tokens = append(tokens, tok)
		default:
			return nil, fmt.Errorf("expression: unexpected character %q at position %d", string(ch), l.pos)
		}
	}
	return tokens, nil
}

func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.input) && unicode.IsSpace(l.input[l.pos]) {
		l.pos++
	}
}

func (l *Lexer) readString(quote rune) (Token, error) {
	l.pos++ // skip opening quote
	var b strings.Builder
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if ch == '\\' && l.pos+1 < len(l.input) {
			next := l.input[l.pos+1]
			if next == quote || next == '\\' {
				b.WriteRune(next)
				l.pos += 2
				continue
			}
		}
		if ch == quote {
			l.pos++ // skip closing quote
			return Token{Type: TokenString, Value: b.String()}, nil
		}
		b.WriteRune(ch)
		l.pos++
	}
	return Token{}, fmt.Errorf("expression: unterminated string starting at position %d", l.pos)
}

func (l *Lexer) readNumber() (Token, error) {
	start := l.pos
	if l.pos < len(l.input) && l.input[l.pos] == '-' {
		l.pos++
	}
	if l.pos >= len(l.input) || !isDigit(l.input[l.pos]) {
		return Token{}, fmt.Errorf("expression: invalid number at position %d", start)
	}
	for l.pos < len(l.input) && isDigit(l.input[l.pos]) {
		l.pos++
	}
	if l.pos < len(l.input) && l.input[l.pos] == '.' {
		l.pos++
		if l.pos >= len(l.input) || !isDigit(l.input[l.pos]) {
			return Token{}, fmt.Errorf("expression: invalid number at position %d", start)
		}
		for l.pos < len(l.input) && isDigit(l.input[l.pos]) {
			l.pos++
		}
	}
	return Token{Type: TokenNumber, Value: string(l.input[start:l.pos])}, nil
}

func (l *Lexer) readIdent() Token {
	start := l.pos
	for l.pos < len(l.input) && isIdentPart(l.input[l.pos]) {
		l.pos++
	}
	value := string(l.input[start:l.pos])
	if value == "true" || value == "false" {
		return Token{Type: TokenBool, Value: value}
	}
	return Token{Type: TokenIdent, Value: value}
}

func isDigit(r rune) bool      { return r >= '0' && r <= '9' }
func isIdentStart(r rune) bool { return unicode.IsLetter(r) || r == '_' }
func isIdentPart(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '.'
}
