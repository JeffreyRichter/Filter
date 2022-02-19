// https://www.youtube.com/watch?v=HxaD_trXwRE
// https://talks.golang.org/2011/lex.slide#7
// https://blog.gopheracademy.com/advent-2014/parsers-lexers/
// http://docs.oasis-open.org/odata/odata/v4.01/cs01/abnf/odata-abnf-construction-rules.txt
// https://github.com/marcak/calc
// https://github.com/benbjohnson/sql-parser
// https://bnfplayground.pauliankline.com/?bnf=%2F*%20(i1%20eq%20%27jeff%27%20and%20(i2%20ne%205%20or%20i3%20ne%20true))%20*%2F%0A%3CConjunction%3E%20%20%20%20%20%20%3A%3A%3D%20%3CParenPropCompare%3E%20%7C%20%3CParenPropCompare%3E%20(%22and%22%20%7C%20%22or%22)%20%3CParenPropCompare%3E%0A%3CParenPropCompare%3E%20%3A%3A%3D%20%22(%22%20%3CPropCompare%3E%20%22)%22%20%7C%20%3CPropCompare%3E%0A%3CPropCompare%3E%20%20%20%20%20%20%3A%3A%3D%20%3CJsonPtr%3E%20%3CCompareOp%3E%20(%3CString%3E%20%7C%20%3CDecimal%3E%20%7C%20%3CBoolean%3E)%0A%3CCompareOp%3E%20%20%20%20%20%20%20%20%3A%3A%3D%20%22eq%22%20%7C%20%22ne%22%20%7C%20%22ge%22%20%7C%20%22gt%22%20%7C%20%22le%22%20%7C%20%22lt%22%0A%3CString%3E%20%20%20%20%20%20%20%20%20%20%20%3A%3A%3D%20%22%27%22%20%3CStringChars%3E*%20%22%27%22%0A%3CStringChars%3E%20%20%20%20%20%20%3A%3A%3D%20(%5BA-Z%5D%20%7C%20%5Ba-z%5D%20%7C%20%5B0-9%5D%20%7C%20%22~%22%20%7C%20%22!%22%20%7C%20%22%40%22%20%7C%20%22%23%22%20%7C%20%22%24%22%20%7C%20%22%25%22%20%7C%20%22%5E%22%20%7C%20%22%26%22%20%7C%20%22*%22%20%7C%20%22(%22%20%7C%20%22)%22%20%7C%20%22-%22%20%7C%20%22_%22%20%7C%20%22%3D%22%20%7C%20%22%2B%22%20%7C%20%22%5B%22%20%7C%20%22%5D%22%20%7C%20%22%7B%22%20%7C%20%22%7D%22%20%7C%20%22%5C%22%20%7C%20%22%3B%22%20%7C%20%22%3A%22%20%7C%20%22%2C%22%20%7C%20%22.%22%20%7C%20%22%2F%22%20%7C%20%22%3C%22%20%7C%20%22%3E%22%20%7C%20%22%3F%22)*%0A%3CBoolean%3E%20%20%20%20%20%20%20%20%20%20%3A%3A%3D%20%22true%22%20%7C%20%22false%22%0A%3CDecimal%3E%20%20%20%20%20%20%20%20%20%20%3A%3A%3D%20%3CInteger%3E%20%22.%22%20%3CInteger%3E%20%7C%20%3CInteger%3E%0A%3CInteger%3E%20%20%20%20%20%20%20%20%20%20%3A%3A%3D%20%3CDigit%3E%20%3CInteger%3E%20%7C%20%3CDigit%3E%0A%3CDigit%3E%20%20%20%20%20%20%20%20%20%20%20%20%3A%3A%3D%20%5B0-9%5D%0A%3CJsonPtr%3E%20%20%20%20%20%20%20%20%20%20%3A%3A%3D%20(%22%2F%22%20%3CRefToken%3E)*%0A%3CRefToken%3E%20%20%20%20%20%20%20%20%20%3A%3A%3D%20(%20%3CUnescaped%3E%20%7C%20%3CEscaped%3E%20)*%0A%3CUnescaped%3E%20%20%20%20%20%20%20%20%3A%3A%3D%20%220%22%0A%2F*%5B%25x00-2E%5D%20%7C%20%5B%25x30-7D%5D%20%7C%20%5B%25x7F-10FFFF%5D*%2F%0A%2F*%20%25x2F%20(%27%2F%27)%20and%20%25x7E%20(%27~%27)%20are%20excluded%20from%20%27unescaped%27%20*%2F%0A%3CEscaped%3E%20%20%20%20%20%20%20%20%20%20%3A%3A%3D%20%22~%22%20(%20%220%22%20%7C%20%221%22%20)%20%20%20%20%20%20%20%20%20%0A%2F*%20%20representing%20%27~%27%20and%20%27%2F%27%2C%20respectively%20*%2F%0A&name=

package main

import (
	"fmt"
	"time"

	"githib.com/JeffreyRichter/filter/filter"
)

func main() {
	json := map[string]any{
		"string": "Jeff",
		"int":    23,
		"float":  3.14,
		"bool":   true,
		"time":   time.Date(1990, time.January, 1, 0, 0, 0, 0, time.UTC),
		"child": map[string]any{
			"childString": "child",
			"childBool":   false,
			"childInt":    42,
		},
	}

	test := "foo ne null or child.childInt eq 42 and time gt time'1989-01-01T00:00:00Z' and (bool eq true and string eq 'Jeffr') and int gt 23 and float le 5"
	f, err := filter.New(test)
	if err != nil {
		fmt.Printf("Filter error: %v\n", err)
	} else {
		r, err := f.Evaluate(json)
		fmt.Printf("Evaluation: %v, Err: %v\n", r, err)
	}
}
