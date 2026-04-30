// Package expression provides a lexer, parser, and evaluator for
// runtime expression functions used in scrutineer YAML test manifests.
//
// Expressions use the syntax ${fn:function_name(arg1, arg2, ...)} where
// the fn: prefix distinguishes them from variable lookups.
package expression

// TokenType classifies lexer tokens.
type TokenType int

const (
	TokenIdent  TokenType = iota // function name or dotted variable reference
	TokenLParen                  // (
	TokenRParen                  // )
	TokenComma                   // ,
	TokenString                  // "quoted" or 'quoted' string literal
	TokenNumber                  // integer or float literal
	TokenBool                    // true or false
)

// Token is a single lexical unit produced by the Lexer.
type Token struct {
	Type  TokenType
	Value string
}
