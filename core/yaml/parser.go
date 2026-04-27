package yaml

import (
	"strings"
)

// NodeType identifies the kind of node in the parsed YAML tree.
type NodeType int

const (
	// ScalarNode represents a scalar value (string, int, float, bool, null).
	ScalarNode NodeType = iota
	// MappingNode represents a YAML mapping (key-value pairs).
	MappingNode
	// SequenceNode represents a YAML sequence (ordered list).
	SequenceNode
)

// String returns a human-readable name for the node type.
func (n NodeType) String() string {
	switch n {
	case ScalarNode:
		return "Scalar"
	case MappingNode:
		return "Mapping"
	case SequenceNode:
		return "Sequence"
	default:
		return "Unknown"
	}
}

// Node represents a node in the parsed YAML tree.
type Node struct {
	Type     NodeType
	Value    string     // for ScalarNode
	Pairs    []KeyValue // for MappingNode (ordered)
	Children []*Node    // for SequenceNode
	Line     int
	Column   int
}

// KeyValue represents a single key-value pair in a mapping.
type KeyValue struct {
	Key   string
	Value *Node
}

// Parser parses a token stream into a YAML node tree.
type Parser struct {
	tokens []Token
	pos    int
}

// Parse parses YAML input bytes and returns the root node.
func Parse(input []byte) (*Node, error) {
	scanner := NewScanner(string(input))
	tokens, err := scanner.ScanAll()
	if err != nil {
		return nil, err
	}

	p := &Parser{tokens: tokens}
	node, err := p.parseDocument()
	if err != nil {
		return nil, err
	}
	return node, nil
}

func (p *Parser) peek() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) next() Token {
	tok := p.peek()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return tok
}

func (p *Parser) parseDocument() (*Node, error) {
	tok := p.peek()
	if tok.Type == TokenEOF {
		return &Node{Type: ScalarNode, Value: "", Line: tok.Line, Column: tok.Column}, nil
	}
	return p.parseNode(-1)
}

func (p *Parser) parseNode(minIndent int) (*Node, error) {
	tok := p.peek()

	switch tok.Type {
	case TokenEOF:
		return &Node{Type: ScalarNode, Value: "", Line: tok.Line, Column: tok.Column}, nil

	case TokenFlowSequenceStart:
		return p.parseFlowSequence()

	case TokenFlowMappingStart:
		return p.parseFlowMapping()

	case TokenMappingKey:
		return p.parseBlockMapping(minIndent)

	case TokenSequenceEntry:
		return p.parseBlockSequence(minIndent)

	case TokenScalar:
		p.next()
		return &Node{Type: ScalarNode, Value: tok.Value, Line: tok.Line, Column: tok.Column}, nil

	case TokenLiteralBlock, TokenFoldedBlock:
		p.next()
		return &Node{Type: ScalarNode, Value: tok.Value, Line: tok.Line, Column: tok.Column}, nil

	default:
		return nil, newParseErrorf(tok.Line, tok.Column, "unexpected token: %s", tok.Type)
	}
}

func (p *Parser) parseBlockMapping(minIndent int) (*Node, error) {
	node := &Node{Type: MappingNode}
	firstTok := p.peek()
	node.Line = firstTok.Line
	node.Column = firstTok.Column
	mapIndent := firstTok.Indent
	firstKey := true

	for {
		tok := p.peek()
		if tok.Type != TokenMappingKey {
			break
		}
		if tok.Indent < mapIndent {
			break
		}
		if tok.Indent > mapIndent {
			// If this is the second key and the first was inline (from sequence entry),
			// adjust mapIndent to this key's indent level since this is the real
			// indentation level of the mapping.
			if firstKey {
				break
			}
			if len(node.Pairs) == 1 && mapIndent <= minIndent {
				mapIndent = tok.Indent
			} else {
				break
			}
		}
		firstKey = false

		p.next() // consume the key token
		key := tok.Key()

		// Parse the value
		val, err := p.parseMappingValue(mapIndent)
		if err != nil {
			return nil, err
		}
		node.Pairs = append(node.Pairs, KeyValue{Key: key, Value: val})
	}

	return node, nil
}

// Key returns the key value from a MappingKey token.
func (t Token) Key() string {
	return t.Value
}

func (p *Parser) parseMappingValue(keyIndent int) (*Node, error) {
	tok := p.peek()

	// Check what follows the key
	switch tok.Type {
	case TokenEOF:
		// empty value
		return &Node{Type: ScalarNode, Value: "", Line: tok.Line, Column: tok.Column}, nil

	case TokenScalar:
		p.next()
		return &Node{Type: ScalarNode, Value: tok.Value, Line: tok.Line, Column: tok.Column}, nil

	case TokenLiteralBlock, TokenFoldedBlock:
		p.next()
		return &Node{Type: ScalarNode, Value: tok.Value, Line: tok.Line, Column: tok.Column}, nil

	case TokenFlowSequenceStart:
		return p.parseFlowSequence()

	case TokenFlowMappingStart:
		return p.parseFlowMapping()

	case TokenMappingKey:
		// Nested mapping - must be more indented
		if tok.Indent > keyIndent {
			return p.parseBlockMapping(keyIndent)
		}
		// Same or less indent: empty value for current key
		return &Node{Type: ScalarNode, Value: "", Line: tok.Line, Column: tok.Column}, nil

	case TokenSequenceEntry:
		// Nested sequence - must be more indented
		if tok.Indent > keyIndent {
			return p.parseBlockSequence(keyIndent)
		}
		return &Node{Type: ScalarNode, Value: "", Line: tok.Line, Column: tok.Column}, nil

	default:
		return &Node{Type: ScalarNode, Value: "", Line: tok.Line, Column: tok.Column}, nil
	}
}

func (p *Parser) parseBlockSequence(minIndent int) (*Node, error) {
	node := &Node{Type: SequenceNode}
	firstTok := p.peek()
	node.Line = firstTok.Line
	node.Column = firstTok.Column
	seqIndent := firstTok.Indent

	for {
		tok := p.peek()
		if tok.Type != TokenSequenceEntry {
			break
		}
		if tok.Indent != seqIndent {
			break
		}
		p.next() // consume "-"

		// The item content starts after "- "
		// Its effective indent for nested content is seqIndent + 2
		itemTok := p.peek()
		var child *Node
		var err error

		switch itemTok.Type {
		case TokenMappingKey:
			// Map item in sequence. The keys should be at seqIndent+2 or more.
			child, err = p.parseBlockMapping(seqIndent)
		case TokenSequenceEntry:
			child, err = p.parseBlockSequence(seqIndent)
		case TokenFlowSequenceStart:
			child, err = p.parseFlowSequence()
		case TokenFlowMappingStart:
			child, err = p.parseFlowMapping()
		case TokenScalar:
			p.next()
			child = &Node{Type: ScalarNode, Value: itemTok.Value, Line: itemTok.Line, Column: itemTok.Column}
		case TokenLiteralBlock, TokenFoldedBlock:
			p.next()
			child = &Node{Type: ScalarNode, Value: itemTok.Value, Line: itemTok.Line, Column: itemTok.Column}
		case TokenEOF:
			child = &Node{Type: ScalarNode, Value: "", Line: itemTok.Line, Column: itemTok.Column}
		default:
			return nil, newParseErrorf(itemTok.Line, itemTok.Column, "unexpected token in sequence: %s", itemTok.Type)
		}
		if err != nil {
			return nil, err
		}
		node.Children = append(node.Children, child)
	}

	return node, nil
}

func (p *Parser) parseFlowSequence() (*Node, error) {
	tok := p.next() // consume '['
	node := &Node{Type: SequenceNode, Line: tok.Line, Column: tok.Column}

	for {
		tok = p.peek()
		if tok.Type == TokenFlowSequenceEnd {
			p.next()
			return node, nil
		}
		if tok.Type == TokenEOF {
			return nil, newParseError(tok.Line, tok.Column, "unterminated flow sequence")
		}
		if tok.Type == TokenFlowComma {
			p.next()
			continue
		}

		child, err := p.parseFlowValue()
		if err != nil {
			return nil, err
		}
		node.Children = append(node.Children, child)
	}
}

func (p *Parser) parseFlowMapping() (*Node, error) {
	tok := p.next() // consume '{'
	node := &Node{Type: MappingNode, Line: tok.Line, Column: tok.Column}

	for {
		tok = p.peek()
		if tok.Type == TokenFlowMappingEnd {
			p.next()
			return node, nil
		}
		if tok.Type == TokenEOF {
			return nil, newParseError(tok.Line, tok.Column, "unterminated flow mapping")
		}
		if tok.Type == TokenFlowComma {
			p.next()
			continue
		}

		// Expect a key
		if tok.Type == TokenMappingKey {
			p.next()
			key := tok.Value
			// Parse value
			valTok := p.peek()
			var val *Node
			var err error
			if valTok.Type == TokenFlowMappingEnd || valTok.Type == TokenFlowComma {
				val = &Node{Type: ScalarNode, Value: "", Line: valTok.Line, Column: valTok.Column}
			} else {
				val, err = p.parseFlowValue()
				if err != nil {
					return nil, err
				}
			}
			node.Pairs = append(node.Pairs, KeyValue{Key: key, Value: val})
		} else if tok.Type == TokenScalar {
			// Unquoted key in flow mapping: "key: value" might appear as scalar
			// This shouldn't happen with our scanner, but handle it
			p.next()
			key := tok.Value
			// Check for colon
			node.Pairs = append(node.Pairs, KeyValue{Key: key, Value: &Node{Type: ScalarNode, Value: ""}})
		} else {
			return nil, newParseErrorf(tok.Line, tok.Column, "unexpected token in flow mapping: %s", tok.Type)
		}
	}
}

func (p *Parser) parseFlowValue() (*Node, error) {
	tok := p.peek()
	switch tok.Type {
	case TokenFlowSequenceStart:
		return p.parseFlowSequence()
	case TokenFlowMappingStart:
		return p.parseFlowMapping()
	case TokenMappingKey:
		// In flow context, a mapping key token means we have a key: value inside
		// This is handled by the flow mapping parser
		p.next()
		return &Node{Type: ScalarNode, Value: tok.Value, Line: tok.Line, Column: tok.Column}, nil
	case TokenScalar:
		p.next()
		return &Node{Type: ScalarNode, Value: tok.Value, Line: tok.Line, Column: tok.Column}, nil
	default:
		return nil, newParseErrorf(tok.Line, tok.Column, "unexpected token in flow value: %s", tok.Type)
	}
}

// nodeToInterface converts a Node tree into Go native types.
func nodeToInterface(n *Node) any {
	if n == nil {
		return nil
	}
	switch n.Type {
	case ScalarNode:
		return interpretScalar(n.Value)
	case MappingNode:
		m := make(map[string]any, len(n.Pairs))
		for _, kv := range n.Pairs {
			m[kv.Key] = nodeToInterface(kv.Value)
		}
		return m
	case SequenceNode:
		s := make([]any, len(n.Children))
		for i, child := range n.Children {
			s[i] = nodeToInterface(child)
		}
		return s
	}
	return nil
}

// interpretScalar converts a scalar string to its typed Go value.
func interpretScalar(s string) any {
	if s == "" {
		return nil
	}

	lower := strings.ToLower(s)

	// Null
	if lower == "null" || lower == "~" {
		return nil
	}

	// Booleans
	if lower == "true" || lower == "yes" {
		return true
	}
	if lower == "false" || lower == "no" {
		return false
	}

	// Integer
	if isInteger(s) {
		return parseInt(s)
	}

	// Float
	if isFloat(s) {
		return parseFloat(s)
	}

	return s
}

func isInteger(s string) bool {
	if len(s) == 0 {
		return false
	}
	start := 0
	if s[0] == '-' || s[0] == '+' {
		start = 1
		if start >= len(s) {
			return false
		}
	}
	for i := start; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func parseInt(s string) int {
	neg := false
	start := 0
	if s[0] == '-' {
		neg = true
		start = 1
	} else if s[0] == '+' {
		start = 1
	}
	n := 0
	for i := start; i < len(s); i++ {
		n = n*10 + int(s[i]-'0')
	}
	if neg {
		return -n
	}
	return n
}

func isFloat(s string) bool {
	if len(s) == 0 {
		return false
	}
	hasDot := false
	hasE := false
	start := 0
	if s[0] == '-' || s[0] == '+' {
		start = 1
		if start >= len(s) {
			return false
		}
	}
	for i := start; i < len(s); i++ {
		if s[i] == '.' {
			if hasDot || hasE {
				return false
			}
			hasDot = true
		} else if s[i] == 'e' || s[i] == 'E' {
			if hasE {
				return false
			}
			hasE = true
			if i+1 < len(s) && (s[i+1] == '+' || s[i+1] == '-') {
				i++
			}
		} else if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return hasDot || hasE
}

func parseFloat(s string) float64 {
	// Simple float parsing
	neg := false
	start := 0
	if s[0] == '-' {
		neg = true
		start = 1
	} else if s[0] == '+' {
		start = 1
	}

	var intPart float64
	i := start
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		intPart = intPart*10 + float64(s[i]-'0')
		i++
	}

	var fracPart float64
	if i < len(s) && s[i] == '.' {
		i++
		div := 10.0
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			fracPart += float64(s[i]-'0') / div
			div *= 10
			i++
		}
	}

	result := intPart + fracPart

	if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
		i++
		expNeg := false
		if i < len(s) && s[i] == '-' {
			expNeg = true
			i++
		} else if i < len(s) && s[i] == '+' {
			i++
		}
		var exp float64
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			exp = exp*10 + float64(s[i]-'0')
			i++
		}
		multiplier := 1.0
		for j := 0; j < int(exp); j++ {
			multiplier *= 10
		}
		if expNeg {
			result /= multiplier
		} else {
			result *= multiplier
		}
	}

	if neg {
		return -result
	}
	return result
}
