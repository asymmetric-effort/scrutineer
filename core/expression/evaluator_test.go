package expression

import "testing"

type mockResolver struct {
	data map[string]any
}

func (m *mockResolver) Resolve(ref string) (any, bool) {
	v, ok := m.data[ref]
	return v, ok
}

func TestEvalStringLit(t *testing.T) {
	r := NewRegistry()
	e := NewEvaluator(r, &mockResolver{})
	val, err := e.Eval(&StringLit{Value: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if val != "hello" {
		t.Errorf("got %v", val)
	}
}

func TestEvalNumberLitInt(t *testing.T) {
	r := NewRegistry()
	e := NewEvaluator(r, &mockResolver{})
	val, err := e.Eval(&NumberLit{Value: 42, IsInt: true})
	if err != nil {
		t.Fatal(err)
	}
	if val != 42 {
		t.Errorf("got %v", val)
	}
}

func TestEvalNumberLitFloat(t *testing.T) {
	r := NewRegistry()
	e := NewEvaluator(r, &mockResolver{})
	val, err := e.Eval(&NumberLit{Value: 3.14, IsInt: false})
	if err != nil {
		t.Fatal(err)
	}
	if val != 3.14 {
		t.Errorf("got %v", val)
	}
}

func TestEvalBoolLit(t *testing.T) {
	r := NewRegistry()
	e := NewEvaluator(r, &mockResolver{})
	val, err := e.Eval(&BoolLit{Value: true})
	if err != nil {
		t.Fatal(err)
	}
	if val != true {
		t.Errorf("got %v", val)
	}
}

func TestEvalVarRef(t *testing.T) {
	r := NewRegistry()
	resolver := &mockResolver{data: map[string]any{"fixture.name": "Alice"}}
	e := NewEvaluator(r, resolver)
	val, err := e.Eval(&VarRef{Path: "fixture.name"})
	if err != nil {
		t.Fatal(err)
	}
	if val != "Alice" {
		t.Errorf("got %v", val)
	}
}

func TestEvalVarRefMiss(t *testing.T) {
	r := NewRegistry()
	e := NewEvaluator(r, &mockResolver{data: map[string]any{}})
	_, err := e.Eval(&VarRef{Path: "missing.var"})
	if err == nil {
		t.Fatal("expected error for unresolved variable")
	}
}

func TestEvalSimpleFunction(t *testing.T) {
	r := DefaultRegistry()
	e := NewEvaluator(r, &mockResolver{})
	val, err := e.Eval(&FuncCall{
		Name: "upper",
		Args: []Expr{&StringLit{Value: "hello"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if val != "HELLO" {
		t.Errorf("got %v", val)
	}
}

func TestEvalNestedFunction(t *testing.T) {
	r := DefaultRegistry()
	e := NewEvaluator(r, &mockResolver{})
	val, err := e.Eval(&FuncCall{
		Name: "concat",
		Args: []Expr{
			&FuncCall{Name: "upper", Args: []Expr{&StringLit{Value: "a"}}},
			&FuncCall{Name: "lower", Args: []Expr{&StringLit{Value: "B"}}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if val != "Ab" {
		t.Errorf("got %v", val)
	}
}

func TestEvalUnknownFunction(t *testing.T) {
	r := NewRegistry()
	e := NewEvaluator(r, &mockResolver{})
	_, err := e.Eval(&FuncCall{Name: "nonexistent", Args: nil})
	if err == nil {
		t.Fatal("expected error for unknown function")
	}
}

func TestEvalStringConvenience(t *testing.T) {
	r := DefaultRegistry()
	resolver := &mockResolver{}
	val, err := EvalString(`upper("world")`, r, resolver)
	if err != nil {
		t.Fatal(err)
	}
	if val != "WORLD" {
		t.Errorf("got %v", val)
	}
}

func TestEvalStringBadLex(t *testing.T) {
	r := DefaultRegistry()
	_, err := EvalString("@bad", r, &mockResolver{})
	if err == nil {
		t.Fatal("expected lex error")
	}
}

func TestEvalStringBadParse(t *testing.T) {
	r := DefaultRegistry()
	_, err := EvalString("", r, &mockResolver{})
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestEvalNestedArgError(t *testing.T) {
	r := DefaultRegistry()
	e := NewEvaluator(r, &mockResolver{})
	// Dotted path that can't be resolved — this should error.
	_, err := e.Eval(&FuncCall{
		Name: "upper",
		Args: []Expr{&VarRef{Path: "missing.var"}},
	})
	if err == nil {
		t.Fatal("expected error from nested arg evaluation")
	}
}

func TestEvalBareIdentAsString(t *testing.T) {
	r := DefaultRegistry()
	e := NewEvaluator(r, &mockResolver{})
	// Bare identifier without dots should be treated as string literal.
	val, err := e.Eval(&VarRef{Path: "alpha"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "alpha" {
		t.Errorf("got %v, want alpha", val)
	}
}

// fakeExpr is a custom Expr type unknown to the evaluator, used to test the
// default/unknown node type branch.
type fakeExpr struct{}

func (*fakeExpr) exprNode() {}

func TestEvalUnknownNodeType(t *testing.T) {
	r := NewRegistry()
	e := NewEvaluator(r, &mockResolver{})
	_, err := e.Eval(&fakeExpr{})
	if err == nil {
		t.Fatal("expected error for unknown node type")
	}
}

// TestExprNodeMarkers covers the marker methods on all AST node types.
func TestExprNodeMarkers(t *testing.T) {
	// Call each marker method directly on the concrete type to ensure
	// the coverage tool registers them.
	fc := &FuncCall{}
	fc.exprNode()
	sl := &StringLit{}
	sl.exprNode()
	nl := &NumberLit{}
	nl.exprNode()
	bl := &BoolLit{}
	bl.exprNode()
	vr := &VarRef{}
	vr.exprNode()
}
