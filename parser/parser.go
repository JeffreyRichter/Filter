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
	NodeContains   = NodeKind("Contains")
)

// Node represents a node of the filter expression.
type Node struct {
	NodeKind
	Comparison // If Kind == PropCompare, this field is set
	Contains   // If Kind == Contains, this field is set
	Error      error
}

type typeMismatchError struct {
	msg string
}

func (e *typeMismatchError) SetMsg(c Comparison, jsonVal any, t lexer.Token) error {
	return typeMismatchError{msg: fmt.Sprintf("Type mismatch: PropName(%s)='%v' while literal(%s)='%s'", c.PropName, jsonVal, c.Literal.TokenKind, c.Literal.Symbol)}
}

func (e typeMismatchError) Error() string { return e.msg }

// Parser scans tokens finding its nodes
type parser struct {
	tokens []lexer.Token // Tokens being parsed
	pos    int           // current token
	start  int           // start token for a node
	nodes  []Node        // Nodes created from the tokens
}

// next reads the next token or eof
func (p *parser) next() (token lexer.Token) {
	if p.pos >= len(p.tokens) { // No more tokens
		panic("We should never get here because the parser should know to stop after reading TokenEOF or TokenError") //return eof
	}
	p.pos += 1
	return p.tokens[p.pos-1]
}

// backup places the previously read token back
func (p *parser) backup() { p.pos-- }

func (p *parser) acceptOne(tokenKind lexer.TokenKind) bool {
	if tokenKind == p.next().TokenKind {
		return true
	}
	p.backup()
	return false
}

func (p *parser) emit(kind NodeKind, c ...any) {
	n := Node{NodeKind: kind}
	switch kind { // If any type assertion panics, it's an error in this file.
	case NodeComparison:
		n.Comparison = c[0].(Comparison)
	case NodeContains:
		n.Contains = c[0].(Contains)
	}
	showNode(n)
	p.nodes = append(p.nodes, n)
}

func (p *parser) emitError(format string, args ...any) []Node {
	n := Node{NodeKind: NodeError, Error: errors.New(fmt.Sprintf(format, args...))}
	showNode(n)
	return append(p.nodes, n)
}

// GetNodes returns the nodes of filter
func GetNodes(filter string) []Node {
	parens := 0
	p := &parser{tokens: lexer.GetTokens(filter)}
	for {
		switch t := p.next(); t.TokenKind { // Consume next token
		case lexer.TokenError:
			return p.emitError(t.Symbol)

		case lexer.TokenEOF:
			if parens > 0 {
				return p.emitError("Unbalanced parentheses")
			}
			p.emit(NodeEOF)
			return p.nodes

		case lexer.TokenLeftParen:
			parens++
			p.emit(NodeLeftParen)

		case lexer.TokenRightParen:
			parens--
			p.emit(NodeRightParen)

		case lexer.TokenSymbol: // and/or OR the string representing a function/property name
			switch logicalOp := t.Symbol; logicalOp {
			case "and":
				p.emit(NodeAnd)

			case "or":
				p.emit(NodeOr)

			default: // Property or function name
				// if token after symbol is a left paren, this is a function name; else a proprty name
				if p.acceptOne(lexer.TokenLeftParen) { // Function name
					switch t.Symbol { // Do we recogize this function?
					case "contains": // contains function
						propName := p.next()
						if !p.acceptOne(lexer.TokenComma) { // Is there a comma?
							return p.emitError("Expected ',' after property name: %s", propName.Symbol)
						}
						literal := p.next()
						if literal.TokenKind == lexer.TokenEOF {
							return p.emitError("Expected literal after ','")
						}
						if !p.acceptOne(lexer.TokenRightParen) { // Is there a right paren?
							return p.emitError("Expected ')' after literal: %s", literal.Symbol)
						}
						p.emit(NodeContains, Contains{propName.Symbol, literal})
						continue

					default: // Unrecognized function name
						return p.emitError("Unrecognized function name: %s", t.Symbol)
					}
				}

				// Not a function; we assume a property name for a logical comparison operation
				propName := t.Symbol // Not a function; this is a property name
				op := p.next()
				if op.TokenKind != lexer.TokenSymbol {
					return p.emitError("Expected comparison operator after property name: %s", propName)
				}
				isCompareOp := func(s string) bool {
					n := strings.Index("eqneleltgegt", s)
					return (n >= 0) && ((n % 2) == 0)
				}
				if !isCompareOp(op.Symbol) {
					return p.emitError("Invalid comparison operator (%s)", op)
				}
				literal := p.next()
				if literal.TokenKind == lexer.TokenEOF {
					return p.emitError("Expected literal after comparison operator: %s", op)
				}
				p.emit(NodeComparison, Comparison{
					PropName: propName,
					Op:       CompareOp(op.Symbol),
					Literal:  literal})
			}
		}
	}
	//panic("lexer didn't return EOF or Error") // We should never get here
}
