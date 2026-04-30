package expression

import "testing"

func mustTokenize(t *testing.T, input string) []Token {
	t.Helper()
	tokens, err := NewLexer(input).Tokenize()
	if err != nil {
		t.Fatalf("tokenize %q: %v", input, err)
	}
	return tokens
}

func TestParseSimpleCall(t *testing.T) {
	expr, err := Parse(mustTokenize(t, "random_int(1, 100)"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	fc, ok := expr.(*FuncCall)
	if !ok {
		t.Fatalf("expected FuncCall, got %T", expr)
	}
	if fc.Name != "random_int" {
		t.Errorf("name = %q", fc.Name)
	}
	if len(fc.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(fc.Args))
	}
	n1, ok := fc.Args[0].(*NumberLit)
	if !ok || n1.Value != 1 || !n1.IsInt {
		t.Errorf("arg[0] = %+v", fc.Args[0])
	}
	n2, ok := fc.Args[1].(*NumberLit)
	if !ok || n2.Value != 100 || !n2.IsInt {
		t.Errorf("arg[1] = %+v", fc.Args[1])
	}
}

func TestParseNestedCall(t *testing.T) {
	expr, err := Parse(mustTokenize(t, `concat(upper(fixture.name), "suffix")`))
	if err != nil {
		t.Fatal(err)
	}
	fc := expr.(*FuncCall)
	if fc.Name != "concat" || len(fc.Args) != 2 {
		t.Fatalf("concat: %+v", fc)
	}
	inner, ok := fc.Args[0].(*FuncCall)
	if !ok || inner.Name != "upper" {
		t.Errorf("arg[0] = %T", fc.Args[0])
	}
	varRef, ok := inner.Args[0].(*VarRef)
	if !ok || varRef.Path != "fixture.name" {
		t.Errorf("inner arg = %+v", inner.Args[0])
	}
	str, ok := fc.Args[1].(*StringLit)
	if !ok || str.Value != "suffix" {
		t.Errorf("arg[1] = %+v", fc.Args[1])
	}
}

func TestParseNoArgs(t *testing.T) {
	expr, err := Parse(mustTokenize(t, "uuid()"))
	if err != nil {
		t.Fatal(err)
	}
	fc := expr.(*FuncCall)
	if fc.Name != "uuid" || len(fc.Args) != 0 {
		t.Errorf("uuid: %+v", fc)
	}
}

func TestParseVarRefArg(t *testing.T) {
	expr, err := Parse(mustTokenize(t, "env(fixture.key_name)"))
	if err != nil {
		t.Fatal(err)
	}
	fc := expr.(*FuncCall)
	if fc.Name != "env" || len(fc.Args) != 1 {
		t.Fatalf("env: %+v", fc)
	}
	vr, ok := fc.Args[0].(*VarRef)
	if !ok || vr.Path != "fixture.key_name" {
		t.Errorf("arg = %+v", fc.Args[0])
	}
}

func TestParseBoolArg(t *testing.T) {
	expr, err := Parse(mustTokenize(t, "fn(true, false)"))
	if err != nil {
		t.Fatal(err)
	}
	fc := expr.(*FuncCall)
	b1, ok := fc.Args[0].(*BoolLit)
	if !ok || b1.Value != true {
		t.Errorf("arg[0] = %+v", fc.Args[0])
	}
	b2, ok := fc.Args[1].(*BoolLit)
	if !ok || b2.Value != false {
		t.Errorf("arg[1] = %+v", fc.Args[1])
	}
}

func TestParseFloatArg(t *testing.T) {
	expr, err := Parse(mustTokenize(t, "fn(3.14)"))
	if err != nil {
		t.Fatal(err)
	}
	fc := expr.(*FuncCall)
	n, ok := fc.Args[0].(*NumberLit)
	if !ok || n.IsInt || n.Value != 3.14 {
		t.Errorf("arg = %+v", fc.Args[0])
	}
}

func TestParseMissingRParen(t *testing.T) {
	_, err := Parse(mustTokenize(t, "fn(1, 2"))
	if err == nil {
		t.Fatal("expected error for missing )")
	}
}

func TestParseEmpty(t *testing.T) {
	_, err := Parse([]Token{})
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestParseTrailingTokens(t *testing.T) {
	_, err := Parse(mustTokenize(t, "uuid() extra"))
	if err == nil {
		t.Fatal("expected error for trailing tokens")
	}
}

func TestParseUnexpectedToken(t *testing.T) {
	_, err := Parse([]Token{{Type: TokenComma, Value: ","}})
	if err == nil {
		t.Fatal("expected error for unexpected token")
	}
}

func TestParseBadComma(t *testing.T) {
	_, err := Parse(mustTokenize(t, "fn(1 2)"))
	if err == nil {
		t.Fatal("expected error for missing comma")
	}
}

func TestParseVarRefAlone(t *testing.T) {
	expr, err := Parse(mustTokenize(t, "fixture.name"))
	if err != nil {
		t.Fatal(err)
	}
	vr, ok := expr.(*VarRef)
	if !ok || vr.Path != "fixture.name" {
		t.Errorf("expected VarRef, got %T", expr)
	}
}

func TestParseStringAlone(t *testing.T) {
	expr, err := Parse(mustTokenize(t, `"hello"`))
	if err != nil {
		t.Fatal(err)
	}
	s, ok := expr.(*StringLit)
	if !ok || s.Value != "hello" {
		t.Errorf("expected StringLit, got %T", expr)
	}
}

func TestParseNumberAlone(t *testing.T) {
	expr, err := Parse(mustTokenize(t, "42"))
	if err != nil {
		t.Fatal(err)
	}
	n, ok := expr.(*NumberLit)
	if !ok || n.Value != 42 || !n.IsInt {
		t.Errorf("expected NumberLit(42, int), got %+v", expr)
	}
}

func TestParseBoolAlone(t *testing.T) {
	expr, err := Parse(mustTokenize(t, "true"))
	if err != nil {
		t.Fatal(err)
	}
	b, ok := expr.(*BoolLit)
	if !ok || !b.Value {
		t.Errorf("expected BoolLit(true), got %+v", expr)
	}
}

func TestParseInvalidFloatToken(t *testing.T) {
	// Directly construct a token with a value that looks like a float but
	// cannot be parsed by strconv.ParseFloat.
	tokens := []Token{{Type: TokenNumber, Value: "1.2.3"}}
	_, err := Parse(tokens)
	if err == nil {
		t.Fatal("expected error for invalid float number token")
	}
}

func TestParseInvalidIntToken(t *testing.T) {
	// Directly construct a token with a value that cannot be parsed as int.
	tokens := []Token{{Type: TokenNumber, Value: "99999999999999999999"}}
	_, err := Parse(tokens)
	if err == nil {
		t.Fatal("expected error for overflowing int number token")
	}
}

func TestParseUnexpectedEndOfInput(t *testing.T) {
	// parseFuncCall consumes '(' then calls parseExpr which finds no tokens.
	tokens := []Token{
		{Type: TokenIdent, Value: "fn"},
		{Type: TokenLParen, Value: "("},
	}
	_, err := Parse(tokens)
	if err == nil {
		t.Fatal("expected error for unexpected end of input")
	}
}

func TestParseFuncCallMissingRParenAfterArg(t *testing.T) {
	// After parsing first arg, tokens run out before seeing ')'.
	tokens := []Token{
		{Type: TokenIdent, Value: "fn"},
		{Type: TokenLParen, Value: "("},
		{Type: TokenNumber, Value: "1"},
	}
	_, err := Parse(tokens)
	if err == nil {
		t.Fatal("expected error for missing ) after arg")
	}
}

func TestParseNegativeNumber(t *testing.T) {
	expr, err := Parse(mustTokenize(t, "fn(-5)"))
	if err != nil {
		t.Fatal(err)
	}
	fc, ok := expr.(*FuncCall)
	if !ok {
		t.Fatalf("expected FuncCall, got %T", expr)
	}
	n, ok := fc.Args[0].(*NumberLit)
	if !ok || n.Value != -5 || !n.IsInt {
		t.Errorf("expected NumberLit(-5, int), got %+v", fc.Args[0])
	}
}

func TestParseNegativeFloat(t *testing.T) {
	expr, err := Parse(mustTokenize(t, "fn(-3.14)"))
	if err != nil {
		t.Fatal(err)
	}
	fc, ok := expr.(*FuncCall)
	if !ok {
		t.Fatalf("expected FuncCall, got %T", expr)
	}
	n, ok := fc.Args[0].(*NumberLit)
	if !ok || n.IsInt || n.Value != -3.14 {
		t.Errorf("expected NumberLit(-3.14, float), got %+v", fc.Args[0])
	}
}
