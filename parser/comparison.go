package parser

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"githib.com/JeffreyRichter/filter/lexer"
)

// CompareOp represents a comparison operator and provides some type safety.
type CompareOp string

type Comparison struct {
	PropName string // Must be an lexer.Ident
	Op       CompareOp
	Literal  lexer.Token
}

func (c Comparison) Evaluate(jsonVal any) (b bool, err error) {
	if c.Literal.Symbol == "null" { // Comparisons to null are a special case
		b, err = c.compareNullProp(jsonVal != nil)
	} else if err != nil {
		return false, err // Property doesn't exist, report the error
	} else {
		switch jv := jsonVal.(type) {
		case bool:
			b, err = c.compareBooleanProp(jv)

		case int, int8, int16, int32, int64:
			b, err = c.compareIntegerProp(reflect.ValueOf(jsonVal).Int())

		case float32, float64:
			b, err = c.compareFloatProp(reflect.ValueOf(jsonVal).Float())

		case string:
			b, err = c.compareStringProp(jv)

		case time.Time:
			b, err = c.compareTimeProp(jv)
		}
		if err != nil {
			if tme, ok := err.(typeMismatchError); ok {
				err = tme.SetMsg(c, jsonVal, c.Literal)
			}
			return false, err
		}
	}
	return b, nil
}

func (c Comparison) compareNullProp(propertyExists bool) (bool, error) {
	if c.Literal.TokenKind != lexer.TokenSymbol || c.Literal.Symbol != "null" {
		panic("Caller shouldn't have called us")
	}
	switch c.Op {
	case "eq":
		return !propertyExists, nil // true if property doesn't exist
	case "ne":
		return propertyExists, nil // true if property exists
	default:
		return false, errors.New(fmt.Sprintf("Invalid operator: '%s'", c.Op))
	}
}

func (c Comparison) compareBooleanProp(v bool) (bool, error) {
	if c.Literal.TokenKind != lexer.TokenSymbol || c.Literal.Symbol != "true" && c.Literal.Symbol != "false" {
		return false, typeMismatchError{}
	}
	switch n := c.Literal.Symbol == "true"; c.Op {
	case "eq":
		return v == n, nil
	case "ne":
		return v != n, nil
	}
	return false, errors.New(fmt.Sprintf("Invalid operator: '%s'", c.Op))
}

func (c Comparison) compareIntegerProp(v int64) (bool, error) {
	if c.Literal.TokenKind != lexer.TokenNumber {
		return false, typeMismatchError{}
	}
	n, err := strconv.ParseInt(c.Literal.Symbol, 10, 64)
	if err != nil {
		var numerr *strconv.NumError
		if !errors.As(err, &numerr) {
			return false, err
		}
		switch numerr.Err {
		case strconv.ErrRange:
			return false, errors.New(fmt.Sprintf("Number out of range: '%s'", c.Literal.Symbol))
		case strconv.ErrSyntax:
			return false, errors.New(fmt.Sprintf("Number has improper syntax: '%s'", c.Literal.Symbol))
		default:
			return false, err
		}
	}

	switch c.Op {
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
	return false, errors.New(fmt.Sprintf("Invalid operator: '%s'", c.Op))
}

func (c Comparison) compareFloatProp(v float64) (bool, error) {
	if c.Literal.TokenKind != lexer.TokenNumber {
		return false, typeMismatchError{}
	}
	n, err := strconv.ParseFloat(c.Literal.Symbol, 64)
	if err != nil {
		return false, err
	}

	switch c.Op {
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
	return false, errors.New(fmt.Sprintf("Invalid operator: '%s'", c.Op))
}

func isSymbolSurroundedBy(t lexer.Token, prefix, suffix string) (bool, string) {
	b := t.TokenKind == lexer.TokenSymbol &&
		strings.HasPrefix(t.Symbol, prefix) &&
		strings.HasSuffix(t.Symbol, suffix)
	if !b {
		return false, ""
	}
	return b, t.Symbol[len(prefix) : len(t.Symbol)-len(suffix)] // Remove prefix/suffix
}

func (c Comparison) compareStringProp(v string) (bool, error) {
	ok, lit := isSymbolSurroundedBy(c.Literal, "'", "'")
	if !ok {
		return false, typeMismatchError{}
	}
	n := strings.Compare(v, lit)
	switch c.Op {
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
	return false, errors.New(fmt.Sprintf("Invalid operator: '%s'", c.Op))
}

func (c Comparison) compareTimeProp(v time.Time) (bool, error) {
	ok, lit := isSymbolSurroundedBy(c.Literal, "time'", "'")
	if !ok {
		return false, typeMismatchError{}
	}
	n, err := time.Parse(time.RFC3339, lit)
	if err != nil {
		return false, err
	}
	switch c.Op {
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
	return false, errors.New(fmt.Sprintf("Invalid operator: '%s'", c.Op))
}
