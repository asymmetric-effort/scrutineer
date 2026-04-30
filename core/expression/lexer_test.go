package expression

import "testing"

func TestLexSimpleFunction(t *testing.T) {
	tokens, err := NewLexer("random_string(alphanumeric, 1, 10)").Tokenize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// random_string ( alphanumeric , 1 , 10 )
	expected := []struct {
		typ TokenType
		val string
	}{
		{TokenIdent, "random_string"},
		{TokenLParen, "("},
		{TokenIdent, "alphanumeric"},
		{TokenComma, ","},
		{TokenNumber, "1"},
		{TokenComma, ","},
		{TokenNumber, "10"},
		{TokenRParen, ")"},
	}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d: %+v", len(expected), len(tokens), tokens)
	}
	for i, exp := range expected {
		if tokens[i].Type != exp.typ || tokens[i].Value != exp.val {
			t.Errorf("token[%d] = {%d, %q}, want {%d, %q}", i, tokens[i].Type, tokens[i].Value, exp.typ, exp.val)
		}
	}
}

func TestLexNoArgs(t *testing.T) {
	tokens, err := NewLexer("uuid()").Tokenize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 3 {
		t.Fatalf("expected 3 tokens, got %d", len(tokens))
	}
	if tokens[0].Value != "uuid" || tokens[1].Value != "(" || tokens[2].Value != ")" {
		t.Errorf("tokens = %+v", tokens)
	}
}

func TestLexNestedFunction(t *testing.T) {
	tokens, err := NewLexer(`concat(upper(fixture.name), "-suffix")`).Tokenize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// concat ( upper ( fixture.name ) , "-suffix" ) = 9 tokens
	if len(tokens) != 9 {
		t.Fatalf("expected 9 tokens, got %d: %+v", len(tokens), tokens)
	}
	if tokens[0].Value != "concat" {
		t.Errorf("token[0] = %q", tokens[0].Value)
	}
	if tokens[4].Value != "fixture.name" {
		t.Errorf("token[4] = %q, want fixture.name", tokens[4].Value)
	}
	if tokens[7].Type != TokenString || tokens[7].Value != "-suffix" {
		t.Errorf("token[7] = {%d, %q}", tokens[7].Type, tokens[7].Value)
	}
}

func TestLexStringLiterals(t *testing.T) {
	// Double-quoted
	tokens, err := NewLexer(`"hello world"`).Tokenize()
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 1 || tokens[0].Type != TokenString || tokens[0].Value != "hello world" {
		t.Errorf("double-quoted: %+v", tokens)
	}

	// Single-quoted
	tokens, err = NewLexer(`'hello'`).Tokenize()
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 1 || tokens[0].Type != TokenString || tokens[0].Value != "hello" {
		t.Errorf("single-quoted: %+v", tokens)
	}

	// Escaped quote
	tokens, err = NewLexer(`"say \"hi\""`).Tokenize()
	if err != nil {
		t.Fatal(err)
	}
	if tokens[0].Value != `say "hi"` {
		t.Errorf("escaped: %q", tokens[0].Value)
	}
}

func TestLexNumbers(t *testing.T) {
	tests := []struct {
		input string
		value string
	}{
		{"42", "42"},
		{"3.14", "3.14"},
		{"-7", "-7"},
		{"-0.5", "-0.5"},
	}
	for _, tt := range tests {
		tokens, err := NewLexer(tt.input).Tokenize()
		if err != nil {
			t.Errorf("input %q: %v", tt.input, err)
			continue
		}
		if len(tokens) != 1 || tokens[0].Type != TokenNumber || tokens[0].Value != tt.value {
			t.Errorf("input %q: got %+v", tt.input, tokens)
		}
	}
}

func TestLexBooleans(t *testing.T) {
	tokens, err := NewLexer("true, false").Tokenize()
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 3 {
		t.Fatalf("expected 3 tokens, got %d", len(tokens))
	}
	if tokens[0].Type != TokenBool || tokens[0].Value != "true" {
		t.Errorf("token[0] = %+v", tokens[0])
	}
	if tokens[2].Type != TokenBool || tokens[2].Value != "false" {
		t.Errorf("token[2] = %+v", tokens[2])
	}
}

func TestLexVarRef(t *testing.T) {
	tokens, err := NewLexer("fixture.user.name").Tokenize()
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 1 || tokens[0].Type != TokenIdent || tokens[0].Value != "fixture.user.name" {
		t.Errorf("var ref: %+v", tokens)
	}
}

func TestLexEmpty(t *testing.T) {
	tokens, err := NewLexer("").Tokenize()
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 0 {
		t.Errorf("expected 0 tokens, got %d", len(tokens))
	}
}

func TestLexUnterminatedString(t *testing.T) {
	_, err := NewLexer(`"hello`).Tokenize()
	if err == nil {
		t.Fatal("expected error for unterminated string")
	}
}

func TestLexInvalidChar(t *testing.T) {
	_, err := NewLexer("@").Tokenize()
	if err == nil {
		t.Fatal("expected error for invalid character")
	}
}

func TestLexInvalidNumber(t *testing.T) {
	_, err := NewLexer("-").Tokenize()
	if err == nil {
		t.Fatal("expected error for bare minus")
	}
}

func TestLexTrailingDot(t *testing.T) {
	_, err := NewLexer("3.").Tokenize()
	if err == nil {
		t.Fatal("expected error for trailing dot in number")
	}
}
