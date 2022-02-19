package filter

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"githib.com/JeffreyRichter/filter/collections"
	"githib.com/JeffreyRichter/filter/lexer"
	"githib.com/JeffreyRichter/filter/parser"
)

//var showNode = func(node parser.Node) { fmt.Printf("%#v\n", node) }
var showNode = func(node parser.Node) {}

// Filter is a parsed filter created by New.
type Filter []parser.Node

// New parses a filter string and returns a Filter.
// Here's an example of a filter string:
//    (name eq 'Jeff' and age gt 30) or (student eq true and semester.gpa gt 3.5) and graduated gt time'2020-01-01'
// A logical operation is one of: and, or; or has lower precedence: A && B || C && D means (A && B) || (C && D)
// A comparison operation is one of: eq, ne, gt, ge, lt, le.
// A JSON property name is a string literal; use a period to step into child objects (ex: gpa is a child of semester).
// A literal value (after a comparison operator) can be:
//   boolean: true | false
//   integer: (+|-) <digits> -- no decimal point
//   float:   (+|-) <digits> . <digits>
//   string:  '<alphanumeric characters>'
//   time:    time'<rfc3339 time>'
//   null     (represents the precense (ne)/absense(eq) of a property)
func New(filter string) (Filter, error) {
	parseNodes, err := inFixToPostFix(parser.GetNodes(filter))
	if err != nil {
		return nil, err
	}
	for _, n := range parseNodes {
		showNode(n)
		if err := n.Error; err != nil {
			return nil, err
		}
	}
	return Filter(parseNodes), nil
}

// Evaluate applies the filter to the value in map m.
// The each of the map's values must be one of: bool, integer, float, string, time, or a child map (arrays are not supported).
func (f Filter) Evaluate(m map[string]any) (result bool, err error) {
	evalStack := collections.Stack[bool]{}
	for _, node := range f {
		switch node.NodeKind {
		case parser.NodeAnd:
			b1, b2 := evalStack.Pop(), evalStack.Pop()
			evalStack.Push(b1 && b2)

		case parser.NodeOr:
			b1, b2 := evalStack.Pop(), evalStack.Pop()
			evalStack.Push(b1 || b2)

		case parser.NodeComparison:
			// Evaluate this node and push it on the stack
			nc, b := node.Comparison, false
			jsonVal, err := getPropValue(nc.PropName, m) // Try to find property value
			if nc.Literal.Symbol == "null" {             // Comparisons to null are a special case
				b, err = compareNullProp(err == nil, nc.Op, nc.Literal)
			} else if err != nil {
				return false, err // Property doesn't exist, report the error
			} else {
				switch jv := jsonVal.(type) {
				case bool:
					b, err = compareBooleanProp(jv, nc.Op, nc.Literal)

				case int, int8, int16, int32, int64:
					b, err = compareIntegerProp(reflect.ValueOf(jsonVal).Int(), nc.Op, nc.Literal)

				case float32, float64:
					b, err = compareFloatProp(reflect.ValueOf(jsonVal).Float(), nc.Op, nc.Literal)

				case string:
					b, err = compareStringProp(jv, nc.Op, nc.Literal)

				case time.Time:
					b, err = compareTimeProp(jv, nc.Op, nc.Literal)
				}
				if err != nil {
					if tme, ok := err.(typeMismatchError); ok {
						err = tme.SetMsg(nc, jsonVal, node.Literal)
					}
					return false, err
				}
			}
			evalStack.Push(b)
		}
	}
	return evalStack.Pop(), nil // Retun the last (and only) value on the stack
}

type propertyError struct {
	msg string
}

func (e propertyError) Error() string {
	return e.msg
}

func getPropValue(propName string, json map[string]any) (any, error) {
	var jsonVal any = json

	for _, pn := range strings.Split(propName, ".") {
		if jv, ok := jsonVal.(map[string]any); !ok {
			// This is not a JSON object; so we can't walk to a child property
			return nil, propertyError{msg: fmt.Sprintf("Property has no children: '%v'", jsonVal)}
		} else {
			if jv, ok := jv[pn]; !ok {
				return nil, propertyError{msg: fmt.Sprintf("Property '%s' not found in '%v'", pn, jsonVal)}
			} else {
				jsonVal = jv // Walk into the child property
			}
		}
	}
	return jsonVal, nil
}

type typeMismatchError struct {
	msg string
}

func (e *typeMismatchError) SetMsg(c parser.Comparison, jsonVal any, t lexer.Token) error {
	return typeMismatchError{msg: fmt.Sprintf("Type mismatch: PropName(%s)='%v' while literal(%s)='%s'", c.PropName, jsonVal, c.Literal.TokenKind, c.Literal.Symbol)}
}

func (e typeMismatchError) Error() string { return e.msg }

func compareNullProp(propertyExists bool, op parser.CompareOp, t lexer.Token) (bool, error) {
	if t.TokenKind != lexer.TokenSymbol || t.Symbol != "null" {
		panic("Caller shouldn't have called us")
	}
	switch op {
	case "eq":
		return !propertyExists, nil // true if property doesn't exist
	case "ne":
		return propertyExists, nil // true if property exists
	default:
		return false, errors.New(fmt.Sprintf("Invalid operator: '%s'", op))
	}
}

func compareBooleanProp(v bool, op parser.CompareOp, t lexer.Token) (bool, error) {
	if t.TokenKind != lexer.TokenSymbol || t.Symbol != "true" && t.Symbol != "false" {
		return false, typeMismatchError{}
	}
	switch n := t.Symbol == "true"; op {
	case "eq":
		return v == n, nil
	case "ne":
		return v != n, nil
	}
	return false, errors.New(fmt.Sprintf("Invalid operator: '%s'", op))
}

func compareIntegerProp(v int64, op parser.CompareOp, t lexer.Token) (bool, error) {
	if t.TokenKind != lexer.TokenNumber {
		return false, typeMismatchError{}
	}
	n, err := strconv.ParseInt(t.Symbol, 10, 64)
	if err != nil {
		var numerr *strconv.NumError
		if !errors.As(err, &numerr) {
			return false, err
		}
		switch numerr.Err {
		case strconv.ErrRange:
			return false, errors.New(fmt.Sprintf("Number out of range: '%s'", t.Symbol))
		case strconv.ErrSyntax:
			return false, errors.New(fmt.Sprintf("Number has improper syntax: '%s'", t.Symbol))
		default:
			return false, err
		}
	}

	switch op {
	case "eq":
		return v == n, nil
	case "ne":
		return v != n, nil
	case "gt":
		return v > n, nil
	case "ge":
		return v >= n, nil
	case "lt":
		return v < n, nil
	case "le":
		return v <= n, nil
	}
	return false, errors.New(fmt.Sprintf("Invalid operator: '%s'", op))
}

func compareFloatProp(v float64, op parser.CompareOp, t lexer.Token) (bool, error) {
	if t.TokenKind != lexer.TokenNumber {
		return false, typeMismatchError{}
	}
	n, err := strconv.ParseFloat(t.Symbol, 64)
	if err != nil {
		return false, err
	}

	switch op {
	case "eq":
		return v == n, nil
	case "ne":
		return v != n, nil
	case "gt":
		return v > n, nil
	case "ge":
		return v >= n, nil
	case "lt":
		return v < n, nil
	case "le":
		return v <= n, nil
	}
	return false, errors.New(fmt.Sprintf("Invalid operator: '%s'", op))
}

func compareStringProp(v string, op parser.CompareOp, t lexer.Token) (bool, error) {
	if t.TokenKind != lexer.TokenSymbol || t.Symbol[0] != '\'' || t.Symbol[len(t.Symbol)-1] != '\'' {
		return false, typeMismatchError{}
	}
	n := strings.Compare(v, t.Symbol[1:len(t.Symbol)-1]) // Remove apostrophes
	switch op {
	case "eq":
		return n == 0, nil
	case "ne":
		return n != 0, nil
	case "gt":
		return n == 1, nil
	case "ge":
		return n >= 0, nil
	case "lt":
		return n == -1, nil
	case "le":
		return n <= 0, nil
	}
	return false, errors.New(fmt.Sprintf("Invalid operator: '%s'", op))
}

func compareTimeProp(v time.Time, op parser.CompareOp, t lexer.Token) (bool, error) {
	if t.TokenKind != lexer.TokenSymbol || len(t.Symbol) < 6 || !strings.HasPrefix(t.Symbol, "time'") && t.Symbol[len(t.Symbol)-1] != '\'' {
		return false, typeMismatchError{}
	}

	n, err := time.Parse(time.RFC3339, t.Symbol[5:len(t.Symbol)-1]) // Remove time' prefix & ' suffix
	if err != nil {
		return false, err
	}
	switch op {
	case "eq":
		return v.Equal(n), nil
	case "ne":
		return !v.Equal(n), nil
	case "gt":
		return v.After(n), nil
	case "ge":
		return v.Equal(n) || v.After(n), nil
	case "lt":
		return v.Before(n), nil
	case "le":
		return v.Equal(n) || v.Before(n), nil
	}
	return false, errors.New(fmt.Sprintf("Invalid operator: '%s'", op))
}

// Converts in-fix to post-fix (reverse-polish)
func inFixToPostFix(nodes []parser.Node) ([]parser.Node, error) {
	// This uses the ShuntingYard algorithm
	postfixQueue := collections.Queue[parser.Node]{}
	operatorStack := collections.Stack[parser.Node]{}

NextNode:
	for _, n := range nodes { // Read tokens (in-fix order)
		switch n.NodeKind {
		case parser.NodeEOF:
			break NextNode

		case parser.NodeError:
			postfixQueue.Enqueue(n)
			break NextNode

		case parser.NodeComparison: // If property comparison expression, add to queue
			postfixQueue.Enqueue(n)

		case parser.NodeLeftParen, parser.NodeAnd: // If and/or/left-paren, push on stack
			operatorStack.Push(n)

		case parser.NodeOr: // Or is lower precedence than and, so push on stack
			// If higher precedence operator is on stack, pop it off and add to queue
			for {
				op, ok := operatorStack.Peek()
				if !ok || op.NodeKind != parser.NodeAnd {
					break
				}
				postfixQueue.Enqueue(operatorStack.Pop())
			}
			operatorStack.Push(n) // After moving higher precedence operators to queue, push 'or' on stack

		case parser.NodeRightParen: // If right-paren, pop stack to queue until left-paren (discard both parens)
			for !operatorStack.Empty() {
				if n := operatorStack.Pop(); n.NodeKind != parser.NodeLeftParen {
					postfixQueue.Enqueue(n)
				} else {
					break // Discard both parens and stop
				}
			}
		}
	}
	// No more tokens, pop rest of stack to queue
	for !operatorStack.Empty() {
		postfixQueue.Enqueue(operatorStack.Pop())
	}
	return postfixQueue, nil
}
