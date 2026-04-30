package expression

import (
	"fmt"
	"strings"
)

// Resolver resolves variable references during expression evaluation.
type Resolver interface {
	Resolve(ref string) (any, bool)
}

// Evaluator evaluates parsed expression ASTs using a function registry
// and a variable resolver.
type Evaluator struct {
	registry *Registry
	resolver Resolver
}

// NewEvaluator creates an Evaluator.
func NewEvaluator(reg *Registry, resolver Resolver) *Evaluator {
	return &Evaluator{registry: reg, resolver: resolver}
}

// Eval evaluates an expression and returns the result.
func (e *Evaluator) Eval(expr Expr) (any, error) {
	switch node := expr.(type) {
	case *StringLit:
		return node.Value, nil

	case *NumberLit:
		if node.IsInt {
			return int(node.Value), nil
		}
		return node.Value, nil

	case *BoolLit:
		return node.Value, nil

	case *VarRef:
		val, ok := e.resolver.Resolve(node.Path)
		if !ok {
			// Bare identifiers without dots (e.g., "alpha", "alphanumeric")
			// are treated as string literals when they can't be resolved as
			// variable references. This supports the natural YAML syntax:
			//   ${fn:random_string(alphanumeric, 1, 10)}
			if !strings.Contains(node.Path, ".") {
				return node.Path, nil
			}
			return nil, fmt.Errorf("expression: unresolved variable %q", node.Path)
		}
		return val, nil

	case *FuncCall:
		fn, ok := e.registry.Get(node.Name)
		if !ok {
			return nil, fmt.Errorf("expression: unknown function %q", node.Name)
		}
		args := make([]any, len(node.Args))
		for i, argExpr := range node.Args {
			val, err := e.Eval(argExpr)
			if err != nil {
				return nil, err
			}
			args[i] = val
		}
		return fn(args)

	default:
		return nil, fmt.Errorf("expression: unknown node type %T", expr)
	}
}

// EvalString is a convenience that parses and evaluates an expression string.
// The input is the content after the fn: prefix (e.g. "random_int(1, 100)").
func EvalString(input string, reg *Registry, resolver Resolver) (any, error) {
	lexer := NewLexer(input)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return nil, err
	}
	expr, err := Parse(tokens)
	if err != nil {
		return nil, err
	}
	eval := NewEvaluator(reg, resolver)
	return eval.Eval(expr)
}
