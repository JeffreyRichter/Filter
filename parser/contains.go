package parser

import (
	"strings"

	"githib.com/JeffreyRichter/filter/lexer"
)

type Contains struct {
	PropName string // Must be an lexer.Ident
	Literal  lexer.Token
}

func (c Contains) Evaluate(jsonVal any) (bool, error) {
	ok, lit := isSymbolSurroundedBy(c.Literal, "'", "'")
	if !ok {
		return false, typeMismatchError{}
	}
	return strings.Contains(jsonVal.(string), lit), nil
}
