package parser

import (
	"errors"
	"fmt"
	"strings"

	"githib.com/JeffreyRichter/filter/lexer"
)

//var showNode = func(node Node) { fmt.Printf("%#v\n", node) }
var showNode = func(node Node) {}

// NodeKind represents a lexical token.
type NodeKind string

const (
	NodeError      = NodeKind("Error")
	NodeEOF        = NodeKind("EOF")
	NodeLeftParen  = NodeKind("(")
	NodeRightParen = NodeKind(")")
	NodeAnd        = NodeKind("And")
	NodeOr         = NodeKind("Or")
	NodeComparison = NodeKind("Compare")
)

// Node represents a node of the filter expression.
type Node struct {
	NodeKind
	Comparison // If Kind == PropCompare, this field is set
	Error      error
}

// CompareOp represents a comparison operator and provides some type safety.
type CompareOp string

type Comparison struct {
	PropName string // Must be an lexer.Ident
	Op       CompareOp
	Literal  lexer.Token
}

// GetNodes returns the nodes of filter
func GetNodes(filter string) []Node {
	nodes, parens := []Node{}, 0

	emit := func(kind NodeKind, c ...Comparison) {
		if len(c) == 0 { // If not specified, create a blank PropertyComparison
			c = []Comparison{{}}
		}
		n := Node{NodeKind: kind, Comparison: c[0]}
		showNode(n)
		nodes = append(nodes, n)
	}

	emitError := func(format string, args ...any) []Node {
		n := Node{NodeKind: NodeError, Error: errors.New(fmt.Sprintf(format, args...))}
		showNode(n)
		return append(nodes, n)
	}

	tokens := lexer.GetTokens(filter)
	for i := 0; i < len(tokens); i++ {
		switch tokens[i].TokenKind {
		case lexer.TokenError:
			return emitError(tokens[i].Symbol)

		case lexer.TokenEOF:
			if parens > 0 {
				return emitError("Unbalanced parentheses")
			}
			emit(NodeEOF)
			return nodes

		case lexer.TokenLeftParen:
			parens++
			emit(NodeLeftParen)

		case lexer.TokenRightParen:
			parens--
			emit(NodeRightParen)

		case lexer.TokenSymbol: // and/or OR the string representing a property name
			switch logicalOp := tokens[i].Symbol; logicalOp {
			case "and":
				emit(NodeAnd)

			case "or":
				emit(NodeOr)

			default:
				if i++; i >= len(tokens) { // Advance to the comparison operator
					return emitError("Expected comparison operator")
				}
				op := tokens[i].Symbol
				isCompareOp := func(s string) bool {
					n := strings.Index("eqneleltgegt", s)
					return (n >= 0) && ((n % 2) == 0)
				}
				if !isCompareOp(op) {
					return emitError("Invalid comparison operator (%s)", op)
				}
				if i++; i >= len(tokens) { // Advance to the symbol
					return emitError("Expected symbol after comparison operator")
				}
				emit(NodeComparison, Comparison{
					PropName: tokens[i-2].Symbol,
					Op:       CompareOp(op),
					Literal:  tokens[i]})
			}
		}
	}
	panic("lexer didn't return EOF or Error") // We should never get here
}
