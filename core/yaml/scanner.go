package yaml

import (
	"strings"
	"unicode/utf8"
)

// TokenType identifies the kind of token produced by the scanner.
type TokenType int

const (
	// TokenMappingKey is emitted at the start of a mapping key (implicit).
	TokenMappingKey TokenType = iota
	// TokenMappingValue is emitted after the colon in "key: value".
	TokenMappingValue
	// TokenSequenceEntry is emitted for "- " at the start of a sequence item.
	TokenSequenceEntry
	// TokenScalar is a scalar value (string, number, bool, null).
	TokenScalar
	// TokenFlowSequenceStart is '['.
	TokenFlowSequenceStart
	// TokenFlowSequenceEnd is ']'.
	TokenFlowSequenceEnd
	// TokenFlowMappingStart is '{'.
	TokenFlowMappingStart
	// TokenFlowMappingEnd is '}'.
	TokenFlowMappingEnd
	// TokenFlowComma is ',' inside a flow collection.
	TokenFlowComma
	// TokenLiteralBlock is '|' (literal block scalar indicator).
	TokenLiteralBlock
	// TokenFoldedBlock is '>' (folded block scalar indicator).
	TokenFoldedBlock
	// TokenEOF marks the end of input.
	TokenEOF
)

// String returns a human-readable name for the token type.
func (t TokenType) String() string {
	switch t {
	case TokenMappingKey:
		return "MappingKey"
	case TokenMappingValue:
		return "MappingValue"
	case TokenSequenceEntry:
		return "SequenceEntry"
	case TokenScalar:
		return "Scalar"
	case TokenFlowSequenceStart:
		return "FlowSequenceStart"
	case TokenFlowSequenceEnd:
		return "FlowSequenceEnd"
	case TokenFlowMappingStart:
		return "FlowMappingStart"
	case TokenFlowMappingEnd:
		return "FlowMappingEnd"
	case TokenFlowComma:
		return "FlowComma"
	case TokenLiteralBlock:
		return "LiteralBlock"
	case TokenFoldedBlock:
		return "FoldedBlock"
	case TokenEOF:
		return "EOF"
	default:
		return "Unknown"
	}
}

// Token represents a single lexical token from YAML input.
type Token struct {
	Type   TokenType
	Value  string
	Line   int
	Column int
	Indent int // indentation level (number of leading spaces on the line)
}

// Scanner tokenizes YAML input into a stream of tokens.
type Scanner struct {
	input     string
	pos       int
	line      int
	col       int
	tokens    []Token
	flowDepth int // nesting depth of flow collections
}

// NewScanner creates a new Scanner for the given input.
func NewScanner(input string) *Scanner {
	return &Scanner{
		input: input,
		pos:   0,
		line:  1,
		col:   1,
	}
}

// ScanAll tokenizes the entire input and returns the token slice.
func (s *Scanner) ScanAll() ([]Token, error) {
	for {
		tok, err := s.nextToken()
		if err != nil {
			return nil, err
		}
		s.tokens = append(s.tokens, tok)
		if tok.Type == TokenEOF {
			break
		}
	}
	return s.tokens, nil
}

func (s *Scanner) peek() rune {
	if s.pos >= len(s.input) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(s.input[s.pos:])
	return r
}

func (s *Scanner) advance() rune {
	if s.pos >= len(s.input) {
		return 0
	}
	r, size := utf8.DecodeRuneInString(s.input[s.pos:])
	s.pos += size
	if r == '\n' {
		s.line++
		s.col = 1
	} else {
		s.col++
	}
	return r
}

func (s *Scanner) atEnd() bool {
	return s.pos >= len(s.input)
}

func (s *Scanner) skipInlineWhitespace() {
	for !s.atEnd() && (s.peek() == ' ' || s.peek() == '\t') {
		s.advance()
	}
}

func (s *Scanner) nextToken() (Token, error) {
	if s.flowDepth > 0 {
		return s.nextFlowToken()
	}
	return s.nextBlockToken()
}

func (s *Scanner) nextBlockToken() (Token, error) {
	// Skip blank lines, comment-only lines, and CR
	s.skipBlankLines()

	if s.atEnd() {
		return Token{Type: TokenEOF, Line: s.line, Column: s.col}, nil
	}

	// Count indentation at the start of a line
	indent := 0
	if s.col == 1 {
		for !s.atEnd() && s.peek() == ' ' {
			s.advance()
			indent++
		}
		if s.atEnd() {
			return Token{Type: TokenEOF, Line: s.line, Column: s.col}, nil
		}
		if s.peek() == '#' {
			s.skipToEOL()
			return s.nextBlockToken()
		}
		if s.peek() == '\n' || s.peek() == '\r' {
			s.advance()
			return s.nextBlockToken()
		}
	}

	line := s.line
	col := s.col

	ch := s.peek()

	// Flow collection starters
	if ch == '[' {
		s.advance()
		s.flowDepth++
		return Token{Type: TokenFlowSequenceStart, Line: line, Column: col, Indent: indent}, nil
	}
	if ch == '{' {
		s.advance()
		s.flowDepth++
		return Token{Type: TokenFlowMappingStart, Line: line, Column: col, Indent: indent}, nil
	}

	// Sequence entry: "- "
	if ch == '-' && s.peekAt(1) == ' ' {
		s.advance() // skip '-'
		s.advance() // skip ' '
		return Token{Type: TokenSequenceEntry, Line: line, Column: col, Indent: indent}, nil
	}

	// Block scalar indicators
	if ch == '|' || ch == '>' {
		tokType := TokenLiteralBlock
		if ch == '>' {
			tokType = TokenFoldedBlock
		}
		s.advance()
		// skip optional trailing whitespace/comment and newline
		s.skipInlineWhitespace()
		if !s.atEnd() && s.peek() == '#' {
			s.skipToEOL()
		}
		if !s.atEnd() && (s.peek() == '\n' || s.peek() == '\r') {
			s.advance()
		}
		blockContent := s.readBlockScalar(indent, tokType == TokenLiteralBlock)
		return Token{Type: tokType, Value: blockContent, Line: line, Column: col, Indent: indent}, nil
	}

	// Quoted strings
	if ch == '"' || ch == '\'' {
		val, err := s.readQuotedString(ch)
		if err != nil {
			return Token{}, err
		}
		s.skipInlineWhitespace()
		if !s.atEnd() && s.peek() == ':' && s.isColonTerminator() {
			s.advance() // skip ':'
			s.skipInlineWhitespace()
			return Token{Type: TokenMappingKey, Value: val, Line: line, Column: col, Indent: indent}, nil
		}
		return Token{Type: TokenScalar, Value: val, Line: line, Column: col, Indent: indent}, nil
	}

	// Read plain text line, determining if it's key: value or just a scalar
	return s.readBlockLine(indent, line, col)
}

func (s *Scanner) nextFlowToken() (Token, error) {
	// Skip whitespace and newlines in flow context
	for !s.atEnd() {
		ch := s.peek()
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			s.advance()
			continue
		}
		if ch == '#' {
			s.skipToEOL()
			continue
		}
		break
	}

	if s.atEnd() {
		return Token{Type: TokenEOF, Line: s.line, Column: s.col}, nil
	}

	line := s.line
	col := s.col
	ch := s.peek()

	switch ch {
	case ']':
		s.advance()
		s.flowDepth--
		return Token{Type: TokenFlowSequenceEnd, Line: line, Column: col}, nil
	case '}':
		s.advance()
		s.flowDepth--
		return Token{Type: TokenFlowMappingEnd, Line: line, Column: col}, nil
	case ',':
		s.advance()
		return Token{Type: TokenFlowComma, Line: line, Column: col}, nil
	case '[':
		s.advance()
		s.flowDepth++
		return Token{Type: TokenFlowSequenceStart, Line: line, Column: col}, nil
	case '{':
		s.advance()
		s.flowDepth++
		return Token{Type: TokenFlowMappingStart, Line: line, Column: col}, nil
	}

	// Quoted string in flow
	if ch == '"' || ch == '\'' {
		val, err := s.readQuotedString(ch)
		if err != nil {
			return Token{}, err
		}
		s.skipInlineWhitespace()
		if !s.atEnd() && s.peek() == ':' && s.isFlowColonTerminator() {
			s.advance() // skip ':'
			s.skipInlineWhitespace()
			return Token{Type: TokenMappingKey, Value: val, Line: line, Column: col}, nil
		}
		return Token{Type: TokenScalar, Value: val, Line: line, Column: col}, nil
	}

	// Plain scalar in flow context
	return s.readFlowPlain(line, col)
}

func (s *Scanner) readFlowPlain(line, col int) (Token, error) {
	var buf strings.Builder
	for !s.atEnd() {
		ch := s.peek()
		if ch == ',' || ch == ']' || ch == '}' || ch == '[' || ch == '{' || ch == '\n' || ch == '\r' {
			break
		}
		if ch == ':' && s.isFlowColonTerminator() {
			// This is a mapping key
			val := strings.TrimRight(buf.String(), " \t")
			s.advance() // skip ':'
			s.skipInlineWhitespace()
			return Token{Type: TokenMappingKey, Value: val, Line: line, Column: col}, nil
		}
		if ch == '#' && buf.Len() > 0 {
			str := buf.String()
			if len(str) > 0 && str[len(str)-1] == ' ' {
				s.skipToEOL()
				val := strings.TrimRight(str, " \t")
				return Token{Type: TokenScalar, Value: val, Line: line, Column: col}, nil
			}
		}
		buf.WriteRune(ch)
		s.advance()
	}
	val := strings.TrimRight(buf.String(), " \t")
	return Token{Type: TokenScalar, Value: val, Line: line, Column: col}, nil
}

func (s *Scanner) isFlowColonTerminator() bool {
	// In flow context, colon must be followed by space, comma, }, ], newline, or EOF
	nextPos := s.pos + 1
	if nextPos >= len(s.input) {
		return true
	}
	r, _ := utf8.DecodeRuneInString(s.input[nextPos:])
	return r == ' ' || r == ',' || r == '}' || r == ']' || r == '\n' || r == '\r' || r == '\t'
}

func (s *Scanner) readBlockLine(indent, line, col int) (Token, error) {
	// First pass: scan to find if there's a key: separator on this line
	// We need to identify the colon position before consuming anything
	scanPos := s.pos
	colonOffset := -1 // offset from s.pos where colon is
	inQuote := rune(0)

	for scanPos < len(s.input) {
		r, size := utf8.DecodeRuneInString(s.input[scanPos:])
		if r == '\n' || r == '\r' {
			break
		}
		if inQuote != 0 {
			if r == inQuote {
				inQuote = 0
			} else if r == '\\' && inQuote == '"' {
				scanPos += size // skip escape char
			}
			scanPos += size
			continue
		}
		if r == '"' || r == '\'' {
			inQuote = r
			scanPos += size
			continue
		}
		if r == ':' && colonOffset < 0 {
			// Check next char
			nextPos := scanPos + size
			if nextPos >= len(s.input) {
				colonOffset = scanPos - s.pos
				break
			}
			nr, _ := utf8.DecodeRuneInString(s.input[nextPos:])
			if nr == ' ' || nr == '\n' || nr == '\r' || nr == '\t' {
				colonOffset = scanPos - s.pos
				break
			}
		}
		scanPos += size
	}

	if colonOffset >= 0 {
		// This is a mapping key line
		key := strings.TrimRight(s.input[s.pos:s.pos+colonOffset], " \t")
		// Advance past key and colon by moving to the byte position after colon
		targetPos := s.pos + colonOffset + 1 // +1 for the colon byte
		for s.pos < targetPos {
			s.advance()
		}
		s.skipInlineWhitespace()

		return Token{Type: TokenMappingKey, Value: key, Line: line, Column: col, Indent: indent}, nil
	}

	// No colon found: this is a plain scalar
	var buf strings.Builder
	for !s.atEnd() {
		ch := s.peek()
		if ch == '\n' || ch == '\r' {
			break
		}
		if ch == '#' && buf.Len() > 0 {
			str := buf.String()
			if len(str) > 0 && str[len(str)-1] == ' ' {
				s.skipToEOL()
				return Token{Type: TokenScalar, Value: strings.TrimRight(str, " \t"), Line: line, Column: col, Indent: indent}, nil
			}
		}
		buf.WriteRune(ch)
		s.advance()
	}

	val := strings.TrimRight(buf.String(), " \t")
	return Token{Type: TokenScalar, Value: val, Line: line, Column: col, Indent: indent}, nil
}

func (s *Scanner) skipBlankLines() {
	for !s.atEnd() {
		ch := s.peek()
		if ch == '\n' || ch == '\r' {
			s.advance()
			continue
		}
		if ch == ' ' || ch == '\t' {
			// Peek ahead to see if entire line is blank or comment
			saved := s.pos
			savedLine := s.line
			savedCol := s.col
			for !s.atEnd() && (s.peek() == ' ' || s.peek() == '\t') {
				s.advance()
			}
			if s.atEnd() || s.peek() == '\n' || s.peek() == '\r' || s.peek() == '#' {
				if !s.atEnd() && s.peek() == '#' {
					s.skipToEOL()
				}
				continue
			}
			s.pos = saved
			s.line = savedLine
			s.col = savedCol
			break
		}
		if ch == '#' {
			s.skipToEOL()
			continue
		}
		break
	}
}

func (s *Scanner) skipToEOL() {
	for !s.atEnd() && s.peek() != '\n' {
		s.advance()
	}
}

func (s *Scanner) peekAt(n int) rune {
	pos := s.pos
	for i := 0; i < n; i++ {
		if pos >= len(s.input) {
			return 0
		}
		_, size := utf8.DecodeRuneInString(s.input[pos:])
		pos += size
	}
	if pos >= len(s.input) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(s.input[pos:])
	return r
}

func (s *Scanner) isColonTerminator() bool {
	nextPos := s.pos + 1
	if nextPos >= len(s.input) {
		return true
	}
	r, _ := utf8.DecodeRuneInString(s.input[nextPos:])
	return r == ' ' || r == '\n' || r == '\r' || r == '\t'
}

func (s *Scanner) readQuotedString(quote rune) (string, error) {
	line := s.line
	col := s.col
	s.advance() // skip opening quote
	var buf strings.Builder
	for !s.atEnd() {
		ch := s.peek()
		if ch == quote {
			s.advance()
			return buf.String(), nil
		}
		if quote == '"' && ch == '\\' {
			s.advance()
			if s.atEnd() {
				return "", newParseError(line, col, "unterminated escape sequence in quoted string")
			}
			esc := s.peek()
			s.advance()
			switch esc {
			case 'n':
				buf.WriteByte('\n')
			case 't':
				buf.WriteByte('\t')
			case '\\':
				buf.WriteByte('\\')
			case '"':
				buf.WriteByte('"')
			case '/':
				buf.WriteByte('/')
			case 'r':
				buf.WriteByte('\r')
			default:
				buf.WriteByte('\\')
				buf.WriteRune(esc)
			}
			continue
		}
		buf.WriteRune(ch)
		s.advance()
	}
	return "", newParseError(line, col, "unterminated quoted string")
}

func (s *Scanner) readBlockScalar(parentIndent int, literal bool) string {
	blockIndent := -1
	var lines []string

	for !s.atEnd() {
		savedPos := s.pos
		savedLine := s.line
		savedCol := s.col

		lineIndent := 0
		for !s.atEnd() && s.peek() == ' ' {
			s.advance()
			lineIndent++
		}

		// Blank line
		if s.atEnd() || s.peek() == '\n' || s.peek() == '\r' {
			if !s.atEnd() {
				s.advance()
			}
			lines = append(lines, "")
			continue
		}

		if blockIndent < 0 {
			if lineIndent <= parentIndent {
				s.pos = savedPos
				s.line = savedLine
				s.col = savedCol
				return ""
			}
			blockIndent = lineIndent
		}

		if lineIndent < blockIndent {
			s.pos = savedPos
			s.line = savedLine
			s.col = savedCol
			break
		}

		var lineBuf strings.Builder
		for i := 0; i < lineIndent-blockIndent; i++ {
			lineBuf.WriteByte(' ')
		}
		for !s.atEnd() && s.peek() != '\n' && s.peek() != '\r' {
			lineBuf.WriteRune(s.peek())
			s.advance()
		}
		if !s.atEnd() {
			s.advance()
		}
		lines = append(lines, lineBuf.String())
	}

	// Remove trailing empty lines
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	if literal {
		return strings.Join(lines, "\n") + "\n"
	}
	// Folded: join consecutive non-empty lines with spaces.
	// A blank line produces a newline break (paragraph separator).
	var result strings.Builder
	for i, l := range lines {
		if l == "" {
			// Blank line: end the previous paragraph and add a blank line
			result.WriteByte('\n')
			result.WriteByte('\n')
			continue
		}
		if i > 0 && lines[i-1] != "" {
			result.WriteByte(' ')
		}
		result.WriteString(l)
	}
	result.WriteByte('\n')
	return result.String()
}
