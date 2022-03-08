package filter

import (
	"strings"

	"githib.com/JeffreyRichter/filter/collections"
	"githib.com/JeffreyRichter/filter/parser"
)

//var showNode = func(node parser.Node) { fmt.Printf("%#v\n", node) }
var showNode = func(node parser.Node) {}

// Filter is a parsed filter created by New.
type Filter []parser.Node

// New parses a filter string and returns a Filter.
// Here's an example of a filter string:
//    (name eq 'Jeff' and age gt 30) or (student eq true and semester.gpa gt 3.5) and graduated gt time'2020-01-01'
// A logical operation is one of: and, or; or has lower precedence: A and B or C and D means (A and B) or (C and D)
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

		case parser.NodeContains:
			// Evaluate this node and push it on the stack
			jsonVal := getPropValue(node.Contains.PropName, m) // Try to find property value
			if jsonVal == nil {
				return false, err // Property doesn't exist, report the error
			}
			b, err := node.Contains.Evaluate(jsonVal)
			if err != nil {
				return false, err
			}
			evalStack.Push(b)

		case parser.NodeComparison:
			// Evaluate this node and push it on the stack
			b, err := node.Comparison.Evaluate(getPropValue(node.Comparison.PropName, m))
			if err != nil {
				return false, err
			}
			evalStack.Push(b)
		}
	}
	return evalStack.Pop(), nil // Retun the last (and only) value on the stack
}

/*
type propertyError struct {
	msg string
}

func (e propertyError) Error() string {
	return e.msg
}
*/

func getPropValue(propName string, json map[string]any) any {
	var jsonVal any = json

	for _, pn := range strings.Split(propName, ".") {
		if jv, ok := jsonVal.(map[string]any); !ok {
			// This is not a JSON object; so we can't walk to a child property
			return nil // propertyError{msg: fmt.Sprintf("Property has no children: '%v'", jsonVal)}
		} else {
			if jv, ok := jv[pn]; !ok {
				return nil // propertyError{msg: fmt.Sprintf("Property '%s' not found in '%v'", pn, jsonVal)}
			} else {
				jsonVal = jv // Walk into the child property
			}
		}
	}
	return jsonVal
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

		default: // Just enqueue any other bool expression
			postfixQueue.Enqueue(n)
		}
	}
	// No more tokens, pop rest of stack to queue
	for !operatorStack.Empty() {
		postfixQueue.Enqueue(operatorStack.Pop())
	}
	return postfixQueue, nil
}
