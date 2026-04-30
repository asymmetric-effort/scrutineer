package expression

import (
	"fmt"
	"strconv"
	"strings"
)

// parser is a recursive descent parser for expression tokens.
type parser struct {
	tokens []Token
	pos    int
}

// Parse converts a token slice into an expression AST.
// The top-level expression must be a function call.
func Parse(tokens []Token) (Expr, error) {
	if len(tokens) == 0 {
		return nil, fmt.Errorf("expression: empty expression")
	}
	p := &parser{tokens: tokens}
	expr, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if p.pos < len(p.tokens) {
		return nil, fmt.Errorf("expression: unexpected token %q after expression", p.tokens[p.pos].Value)
	}
	return expr, nil
}

func (p *parser) peek() (Token, bool) {
	if p.pos >= len(p.tokens) {
		return Token{}, false
	}
	return p.tokens[p.pos], true
}

func (p *parser) next() Token {
	tok := p.tokens[p.pos]
	p.pos++
	return tok
}

// parseExpr parses a single expression: function call, literal, or variable reference.
func (p *parser) parseExpr() (Expr, error) {
	tok, ok := p.peek()
	if !ok {
		return nil, fmt.Errorf("expression: unexpected end of input")
	}

	switch tok.Type {
	case TokenString:
		p.next()
		return &StringLit{Value: tok.Value}, nil

	case TokenNumber:
		p.next()
		return p.parseNumber(tok.Value)

	case TokenBool:
		p.next()
		return &BoolLit{Value: tok.Value == "true"}, nil

	case TokenIdent:
		return p.parseIdentOrCall()

	default:
		return nil, fmt.Errorf("expression: unexpected token %q", tok.Value)
	}
}

func (p *parser) parseNumber(s string) (Expr, error) {
	if strings.Contains(s, ".") {
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil, fmt.Errorf("expression: invalid number %q: %w", s, err)
		}
		return &NumberLit{Value: v, IsInt: false}, nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("expression: invalid number %q: %w", s, err)
	}
	return &NumberLit{Value: float64(v), IsInt: true}, nil
}

// parseIdentOrCall handles identifiers which may be function calls (if
// followed by '(') or variable references.
func (p *parser) parseIdentOrCall() (Expr, error) {
	ident := p.next() // consume the identifier

	// Check if followed by '(' — if so, it's a function call.
	tok, ok := p.peek()
	if ok && tok.Type == TokenLParen {
		return p.parseFuncCall(ident.Value)
	}

	// Otherwise it's a variable reference (e.g. "fixture.user.name").
	return &VarRef{Path: ident.Value}, nil
}

func (p *parser) parseFuncCall(name string) (Expr, error) {
	p.next() // consume '('

	var args []Expr

	// Check for empty arg list.
	tok, ok := p.peek()
	if ok && tok.Type == TokenRParen {
		p.next() // consume ')'
		return &FuncCall{Name: name, Args: args}, nil
	}

	// Parse first argument.
	arg, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	args = append(args, arg)

	// Parse remaining comma-separated arguments.
	for {
		tok, ok = p.peek()
		if !ok {
			return nil, fmt.Errorf("expression: expected ')' in call to %s", name)
		}
		if tok.Type == TokenRParen {
			p.next()
			break
		}
		if tok.Type != TokenComma {
			return nil, fmt.Errorf("expression: expected ',' or ')' in call to %s, got %q", name, tok.Value)
		}
		p.next() // consume ','

		arg, err = p.parseExpr()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
	}

	return &FuncCall{Name: name, Args: args}, nil
}
