package yaml

import (
	"testing"
)

func TestScanSimpleKeyValue(t *testing.T) {
	input := "name: Alice\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{TokenMappingKey, TokenScalar, TokenEOF})
	if tokens[0].Value != "name" {
		t.Errorf("expected key 'name', got %q", tokens[0].Value)
	}
	if tokens[1].Value != "Alice" {
		t.Errorf("expected value 'Alice', got %q", tokens[1].Value)
	}
}

func TestScanMultipleKeys(t *testing.T) {
	input := "a: 1\nb: 2\nc: 3\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{
		TokenMappingKey, TokenScalar,
		TokenMappingKey, TokenScalar,
		TokenMappingKey, TokenScalar,
		TokenEOF,
	})
}

func TestScanNestedMapping(t *testing.T) {
	input := "parent:\n  child: value\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{
		TokenMappingKey,
		TokenMappingKey, TokenScalar,
		TokenEOF,
	})
	if tokens[0].Value != "parent" {
		t.Errorf("expected 'parent', got %q", tokens[0].Value)
	}
	if tokens[1].Indent != 2 {
		t.Errorf("expected indent 2, got %d", tokens[1].Indent)
	}
}

func TestScanSequence(t *testing.T) {
	input := "- one\n- two\n- three\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{
		TokenSequenceEntry, TokenScalar,
		TokenSequenceEntry, TokenScalar,
		TokenSequenceEntry, TokenScalar,
		TokenEOF,
	})
}

func TestScanFlowSequence(t *testing.T) {
	input := "[a, b, c]\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{
		TokenFlowSequenceStart,
		TokenScalar, TokenFlowComma,
		TokenScalar, TokenFlowComma,
		TokenScalar,
		TokenFlowSequenceEnd,
		TokenEOF,
	})
}

func TestScanFlowMapping(t *testing.T) {
	input := "{a: 1, b: 2}\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{
		TokenFlowMappingStart,
		TokenMappingKey, TokenScalar,
		TokenFlowComma,
		TokenMappingKey, TokenScalar,
		TokenFlowMappingEnd,
		TokenEOF,
	})
}

func TestScanQuotedString(t *testing.T) {
	input := `name: "hello world"` + "\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	if tokens[1].Value != "hello world" {
		t.Errorf("expected 'hello world', got %q", tokens[1].Value)
	}
}

func TestScanQuotedStringWithEscapes(t *testing.T) {
	input := `msg: "line1\nline2\ttab\\slash\""` + "\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expected := "line1\nline2\ttab\\slash\""
	if tokens[1].Value != expected {
		t.Errorf("expected %q, got %q", expected, tokens[1].Value)
	}
}

func TestScanSingleQuotedString(t *testing.T) {
	input := "name: 'hello'\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	if tokens[1].Value != "hello" {
		t.Errorf("expected 'hello', got %q", tokens[1].Value)
	}
}

func TestScanComments(t *testing.T) {
	input := "# This is a comment\nname: value\n# another comment\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{TokenMappingKey, TokenScalar, TokenEOF})
}

func TestScanInlineComment(t *testing.T) {
	input := "name: value # inline comment\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	if tokens[1].Value != "value" {
		t.Errorf("expected 'value', got %q", tokens[1].Value)
	}
}

func TestScanEmptyValue(t *testing.T) {
	input := "key:\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{TokenMappingKey, TokenEOF})
}

func TestScanLiteralBlock(t *testing.T) {
	input := "desc: |\n  line1\n  line2\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{TokenMappingKey, TokenLiteralBlock, TokenEOF})
	if tokens[1].Value != "line1\nline2\n" {
		t.Errorf("expected 'line1\\nline2\\n', got %q", tokens[1].Value)
	}
}

func TestScanFoldedBlock(t *testing.T) {
	input := "desc: >\n  line1\n  line2\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{TokenMappingKey, TokenFoldedBlock, TokenEOF})
	if tokens[1].Value != "line1 line2\n" {
		t.Errorf("expected 'line1 line2\\n', got %q", tokens[1].Value)
	}
}

func TestScanEmptyInput(t *testing.T) {
	tokens, err := NewScanner("").ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 1 || tokens[0].Type != TokenEOF {
		t.Error("expected single EOF token for empty input")
	}
}

func TestScanOnlyComments(t *testing.T) {
	tokens, err := NewScanner("# just a comment\n# another\n").ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 1 || tokens[0].Type != TokenEOF {
		t.Errorf("expected single EOF, got %d tokens", len(tokens))
	}
}

func TestScanQuotedKey(t *testing.T) {
	input := `"special key": value` + "\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{TokenMappingKey, TokenScalar, TokenEOF})
	if tokens[0].Value != "special key" {
		t.Errorf("expected 'special key', got %q", tokens[0].Value)
	}
}

func TestScanUnterminatedQuote(t *testing.T) {
	input := `name: "unterminated`
	_, err := NewScanner(input).ScanAll()
	if err == nil {
		t.Error("expected error for unterminated quote")
	}
}

func TestScanSequenceOfMaps(t *testing.T) {
	input := "- name: a\n- name: b\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{
		TokenSequenceEntry, TokenMappingKey, TokenScalar,
		TokenSequenceEntry, TokenMappingKey, TokenScalar,
		TokenEOF,
	})
}

func TestScanColonInValue(t *testing.T) {
	input := "url: http://example.com\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{TokenMappingKey, TokenScalar, TokenEOF})
	if tokens[0].Value != "url" {
		t.Errorf("expected key 'url', got %q", tokens[0].Value)
	}
	if tokens[1].Value != "http://example.com" {
		t.Errorf("expected 'http://example.com', got %q", tokens[1].Value)
	}
}

func TestScanKeyAtEOF(t *testing.T) {
	input := "key: value"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{TokenMappingKey, TokenScalar, TokenEOF})
}

func TestScanEscapeSlash(t *testing.T) {
	input := `val: "a\/b"` + "\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	if tokens[1].Value != "a/b" {
		t.Errorf("expected 'a/b', got %q", tokens[1].Value)
	}
}

func TestScanEscapeReturn(t *testing.T) {
	input := `val: "a\rb"` + "\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	if tokens[1].Value != "a\rb" {
		t.Errorf("expected 'a\\rb', got %q", tokens[1].Value)
	}
}

func TestScanUnknownEscape(t *testing.T) {
	input := `val: "a\xb"` + "\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	if tokens[1].Value != `a\xb` {
		t.Errorf("expected 'a\\xb', got %q", tokens[1].Value)
	}
}

func TestTokenTypeStrings(t *testing.T) {
	types := []TokenType{
		TokenMappingKey, TokenMappingValue, TokenSequenceEntry, TokenScalar,
		TokenFlowSequenceStart, TokenFlowSequenceEnd, TokenFlowMappingStart,
		TokenFlowMappingEnd, TokenFlowComma, TokenLiteralBlock, TokenFoldedBlock,
		TokenEOF, TokenType(999),
	}
	expected := []string{
		"MappingKey", "MappingValue", "SequenceEntry", "Scalar",
		"FlowSequenceStart", "FlowSequenceEnd", "FlowMappingStart",
		"FlowMappingEnd", "FlowComma", "LiteralBlock", "FoldedBlock",
		"EOF", "Unknown",
	}
	for i, tt := range types {
		if tt.String() != expected[i] {
			t.Errorf("TokenType(%d).String() = %q, want %q", tt, tt.String(), expected[i])
		}
	}
}

func TestScanLiteralBlockWithExtraIndent(t *testing.T) {
	input := "desc: |\n  line1\n    indented\n  line3\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	if tokens[1].Value != "line1\n  indented\nline3\n" {
		t.Errorf("got %q", tokens[1].Value)
	}
}

func TestScanFoldedBlockWithBlankLine(t *testing.T) {
	input := "desc: >\n  para1\n\n  para2\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	if tokens[1].Value != "para1\n\npara2\n" {
		t.Errorf("got %q", tokens[1].Value)
	}
}

func TestScanBlockScalarCommentAfterIndicator(t *testing.T) {
	input := "desc: | # comment\n  content\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	if tokens[1].Value != "content\n" {
		t.Errorf("got %q", tokens[1].Value)
	}
}

func TestScanMultilineQuotedString(t *testing.T) {
	input := "val: \"line1\nline2\"\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	if tokens[1].Value != "line1\nline2" {
		t.Errorf("got %q", tokens[1].Value)
	}
}

func TestScanCarriageReturn(t *testing.T) {
	input := "a: 1\r\nb: 2\r\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{
		TokenMappingKey, TokenScalar,
		TokenMappingKey, TokenScalar,
		TokenEOF,
	})
}

func TestScanQuotedKeyInFlow(t *testing.T) {
	input := `{"key": "value"}` + "\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{
		TokenFlowMappingStart,
		TokenMappingKey, TokenScalar,
		TokenFlowMappingEnd,
		TokenEOF,
	})
	if tokens[1].Value != "key" {
		t.Errorf("expected 'key', got %q", tokens[1].Value)
	}
}

func TestScanQuotedScalarInFlow(t *testing.T) {
	input := `["hello", "world"]` + "\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{
		TokenFlowSequenceStart,
		TokenScalar, TokenFlowComma,
		TokenScalar,
		TokenFlowSequenceEnd,
		TokenEOF,
	})
}

func TestScanNestedFlow(t *testing.T) {
	input := `{a: [1, 2], b: {c: 3}}` + "\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	// {, a:, [, 1, ',', 2, ], ',', b:, {, c:, 3, }, }
	expectTokenTypes(t, tokens, []TokenType{
		TokenFlowMappingStart,
		TokenMappingKey, TokenFlowSequenceStart,
		TokenScalar, TokenFlowComma,
		TokenScalar,
		TokenFlowSequenceEnd, TokenFlowComma,
		TokenMappingKey, TokenFlowMappingStart,
		TokenMappingKey, TokenScalar,
		TokenFlowMappingEnd,
		TokenFlowMappingEnd,
		TokenEOF,
	})
}

func TestScanOnlyWhitespace(t *testing.T) {
	tokens, err := NewScanner("   \n  \n  ").ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 1 || tokens[0].Type != TokenEOF {
		t.Errorf("expected single EOF, got %d tokens", len(tokens))
	}
}

func TestScanBlockScalarEmptyContent(t *testing.T) {
	// Block scalar where next content is at same or lesser indent
	input := "a: |\nb: val\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{
		TokenMappingKey, TokenLiteralBlock,
		TokenMappingKey, TokenScalar,
		TokenEOF,
	})
	if tokens[1].Value != "" {
		t.Errorf("expected empty block scalar, got %q", tokens[1].Value)
	}
}

func TestScanFoldedBlockScalar(t *testing.T) {
	input := "a: >\n  folded\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{
		TokenMappingKey, TokenFoldedBlock, TokenEOF,
	})
}

func TestScanKeyWithColonAtEOF(t *testing.T) {
	input := "key:"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{TokenMappingKey, TokenEOF})
}

func TestScanCommentAfterIndent(t *testing.T) {
	input := "  # indented comment\nkey: val\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{TokenMappingKey, TokenScalar, TokenEOF})
}

func TestScanDashNotSequence(t *testing.T) {
	// Dash followed by non-space should be a scalar, not a sequence entry
	input := "val: -1\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{TokenMappingKey, TokenScalar, TokenEOF})
	if tokens[1].Value != "-1" {
		t.Errorf("expected '-1', got %q", tokens[1].Value)
	}
}

func TestScanFlowCommentInline(t *testing.T) {
	input := "{a: 1} # comment\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	// After }, we're back to block mode. The # is an inline comment.
	hasEnd := false
	for _, tok := range tokens {
		if tok.Type == TokenFlowMappingEnd {
			hasEnd = true
		}
	}
	if !hasEnd {
		t.Error("expected FlowMappingEnd token")
	}
}

func TestScanFlowSequenceMultiline(t *testing.T) {
	input := "[\n  a,\n  b\n]\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{
		TokenFlowSequenceStart,
		TokenScalar, TokenFlowComma,
		TokenScalar,
		TokenFlowSequenceEnd,
		TokenEOF,
	})
}

func TestScanFlowMappingMultiline(t *testing.T) {
	input := "{\n  a: 1,\n  b: 2\n}\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{
		TokenFlowMappingStart,
		TokenMappingKey, TokenScalar,
		TokenFlowComma,
		TokenMappingKey, TokenScalar,
		TokenFlowMappingEnd,
		TokenEOF,
	})
}

func TestScanFlowCommentInsideFlow(t *testing.T) {
	input := "[a # comment\n, b]\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{
		TokenFlowSequenceStart,
		TokenScalar, TokenFlowComma,
		TokenScalar,
		TokenFlowSequenceEnd,
		TokenEOF,
	})
}

func TestScanQuotedStringAsValue(t *testing.T) {
	// Quoted string that is NOT a key
	input := `"just a value"` + "\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{TokenScalar, TokenEOF})
	if tokens[0].Value != "just a value" {
		t.Errorf("expected 'just a value', got %q", tokens[0].Value)
	}
}

func TestScanUnterminatedEscapeInQuote(t *testing.T) {
	input := `"hello\`
	_, err := NewScanner(input).ScanAll()
	if err == nil {
		t.Error("expected error for unterminated escape")
	}
}

func TestScanSingleQuotedKeyInFlow(t *testing.T) {
	input := "{'key': 'val'}\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{
		TokenFlowMappingStart,
		TokenMappingKey, TokenScalar,
		TokenFlowMappingEnd,
		TokenEOF,
	})
}

func TestScanEmptyFlowSequence(t *testing.T) {
	input := "items: []\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{
		TokenMappingKey,
		TokenFlowSequenceStart, TokenFlowSequenceEnd,
		TokenEOF,
	})
}

func TestScanEmptyFlowMapping(t *testing.T) {
	input := "items: {}\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{
		TokenMappingKey,
		TokenFlowMappingStart, TokenFlowMappingEnd,
		TokenEOF,
	})
}

func TestScanBlockScalarEOFTerminated(t *testing.T) {
	input := "desc: |\n  content"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{TokenMappingKey, TokenLiteralBlock, TokenEOF})
}

func TestScanKeyColonOnlyLine(t *testing.T) {
	// Key at end of line with just colon
	input := "key:\nnext: val\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{
		TokenMappingKey,
		TokenMappingKey, TokenScalar,
		TokenEOF,
	})
}

func TestScanQuotedKeyWithColon(t *testing.T) {
	// Quoted key containing a colon
	input := "\"key:name\": value\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{TokenMappingKey, TokenScalar, TokenEOF})
	if tokens[0].Value != "key:name" {
		t.Errorf("expected 'key:name', got %q", tokens[0].Value)
	}
}

func TestScanLineWithTabIndent(t *testing.T) {
	input := "key:\n\t val\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	// Tab is consumed but not counted as indent
	if len(tokens) < 2 {
		t.Fatal("expected at least 2 tokens")
	}
}

func TestScanBlockLineWithQuotedColon(t *testing.T) {
	// Quoted string in a value that contains a colon
	input := "key: \"has: colon\"\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{TokenMappingKey, TokenScalar, TokenEOF})
	if tokens[1].Value != "has: colon" {
		t.Errorf("expected 'has: colon', got %q", tokens[1].Value)
	}
}

func TestScanBlockLineQuotedKey(t *testing.T) {
	// Quoted string as key in readBlockLine (single-quoted)
	input := "'quoted': value\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{TokenMappingKey, TokenScalar, TokenEOF})
}

func TestScanBlockLineNoColon(t *testing.T) {
	// Plain scalar without any colon
	input := "- justvalue\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{TokenSequenceEntry, TokenScalar, TokenEOF})
	if tokens[1].Value != "justvalue" {
		t.Errorf("expected 'justvalue', got %q", tokens[1].Value)
	}
}

func TestScanFlowColonAtEnd(t *testing.T) {
	// Flow colon at end of input
	input := "{a:}"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{
		TokenFlowMappingStart,
		TokenMappingKey,
		TokenFlowMappingEnd,
		TokenEOF,
	})
}

func TestScanBlockScalarNoTrailingNewline(t *testing.T) {
	// Block scalar where content doesn't end with newline
	input := "desc: |\n  line1\n  line2"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{TokenMappingKey, TokenLiteralBlock, TokenEOF})
}

func TestScanBlockScalarTrailingBlanks(t *testing.T) {
	// Block scalar with trailing blank lines that should be stripped
	input := "desc: |\n  content\n\n\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	if tokens[1].Value != "content\n" {
		t.Errorf("got %q", tokens[1].Value)
	}
}

func TestScanBlockScalarLessIndent(t *testing.T) {
	// Block scalar followed by content at lower indent
	input := "a: |\n    deep\nb: val\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{
		TokenMappingKey, TokenLiteralBlock,
		TokenMappingKey, TokenScalar,
		TokenEOF,
	})
	if tokens[1].Value != "deep\n" {
		t.Errorf("got %q", tokens[1].Value)
	}
}

func TestScanFlowEndAtEOF(t *testing.T) {
	// Flow collection where ] is at EOF
	input := "[a]"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{
		TokenFlowSequenceStart,
		TokenScalar,
		TokenFlowSequenceEnd,
		TokenEOF,
	})
}

func TestScanFlowPlainInlineComment(t *testing.T) {
	// Comment inside flow collection on a value
	input := "[val # comment\n]"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, tok := range tokens {
		if tok.Type == TokenScalar && tok.Value == "val" {
			found = true
		}
	}
	if !found {
		t.Error("expected scalar 'val'")
	}
}

func TestScanAdvancePastEnd(t *testing.T) {
	s := NewScanner("")
	r := s.advance()
	if r != 0 {
		t.Errorf("expected 0 rune, got %v", r)
	}
	r = s.peek()
	if r != 0 {
		t.Errorf("expected 0 rune from peek, got %v", r)
	}
}

func TestScanPeekAtBeyondEnd(t *testing.T) {
	s := NewScanner("a")
	r := s.peekAt(5)
	if r != 0 {
		t.Errorf("expected 0, got %v", r)
	}
}

func TestScanReadBlockLineQuotedColon(t *testing.T) {
	// Key containing a quoted segment with colon inside
	// readBlockLine should skip the colon inside quotes
	// key"x:y": value -> key is key"x:y", colon is after the quote
	input := "key\"x:y\": value\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{TokenMappingKey, TokenScalar, TokenEOF})
}

func TestScanReadBlockLineEscapeInQuote(t *testing.T) {
	// Key with escaped quote inside double-quoted segment
	input := "key\"x\\\"y\": value\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{TokenMappingKey, TokenScalar, TokenEOF})
}

func TestScanReadBlockLineSingleQuotedColon(t *testing.T) {
	// Key containing single-quoted segment with colon
	input := "key'x:y': value\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{TokenMappingKey, TokenScalar, TokenEOF})
}

func TestScanColonTerminatorAtEOF(t *testing.T) {
	s := NewScanner("key:")
	s.pos = 3 // at ':'
	if !s.isColonTerminator() {
		t.Error("expected true for colon at end of input")
	}
}

func TestScanIndentedLineEndingInComment(t *testing.T) {
	// Indented line that is just a comment
	input := "key: val\n  # comment\nnext: val2\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{
		TokenMappingKey, TokenScalar,
		TokenMappingKey, TokenScalar,
		TokenEOF,
	})
}

func TestScanIndentedBlankLine(t *testing.T) {
	// Line with only spaces
	input := "key: val\n   \nnext: val2\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{
		TokenMappingKey, TokenScalar,
		TokenMappingKey, TokenScalar,
		TokenEOF,
	})
}

func TestScanIndentedEOF(t *testing.T) {
	// File ends with spaces after content
	input := "key: val\n  "
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	expectTokenTypes(t, tokens, []TokenType{TokenMappingKey, TokenScalar, TokenEOF})
}

func TestScanFlowColonTerminatorEOF(t *testing.T) {
	s := NewScanner("{a:}")
	s.pos = 2 // at ':'
	s.flowDepth = 1
	if !s.isFlowColonTerminator() {
		t.Error("expected true for flow colon at end")
	}
}

func TestScanFlowQuotedStringUnterminatedInFlow(t *testing.T) {
	input := `["unterminated`
	_, err := NewScanner(input).ScanAll()
	if err == nil {
		t.Error("expected error for unterminated quote in flow")
	}
}

func TestScanFlowCommentInToken(t *testing.T) {
	// Comment inside flow after whitespace
	input := "{ # comment\na: 1}\n"
	tokens, err := NewScanner(input).ScanAll()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, tok := range tokens {
		if tok.Type == TokenMappingKey && tok.Value == "a" {
			found = true
		}
	}
	if !found {
		t.Error("expected to find key 'a'")
	}
}

func TestScanPeekAtEdge(t *testing.T) {
	s := NewScanner("ab")
	// peekAt(0) should return 'a'
	r := s.peekAt(0)
	if r != 'a' {
		t.Errorf("expected 'a', got %v", r)
	}
	// peekAt(1) should return 'b'
	r = s.peekAt(1)
	if r != 'b' {
		t.Errorf("expected 'b', got %v", r)
	}
	// peekAt(2) should return 0
	r = s.peekAt(2)
	if r != 0 {
		t.Errorf("expected 0, got %v", r)
	}
}

func expectTokenTypes(t *testing.T, tokens []Token, expected []TokenType) {
	t.Helper()
	if len(tokens) != len(expected) {
		types := make([]string, len(tokens))
		for i, tok := range tokens {
			types[i] = tok.Type.String()
		}
		t.Fatalf("expected %d tokens, got %d: %v", len(expected), len(tokens), types)
	}
	for i, tok := range tokens {
		if tok.Type != expected[i] {
			t.Errorf("token %d: expected %s, got %s (value=%q)", i, expected[i], tok.Type, tok.Value)
		}
	}
}
