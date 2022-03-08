package lexer

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

//var showToken = func(t Token) { fmt.Printf("%15s  \"%s\"\n", t.TokenKind, t.Symbol) }

var showToken = func(t Token) {}

// TokenKind represents a lexical token.
type TokenKind string

const (
	TokenError      = TokenKind("Error")
	TokenEOF        = TokenKind("EOF")
	TokenWhitespace = TokenKind("Whitespace")
	TokenLeftParen  = TokenKind("(")
	TokenRightParen = TokenKind(")")
	TokenComma      = TokenKind("Comma")
	TokenSymbol     = TokenKind("Symbol")
	TokenNumber     = TokenKind("Number")
)

// Token represents a lexical token.
type Token struct {
	TokenKind        // The kind of token
	Symbol    string // The value of the token
	Error     error  // The error, if any
}

// lexer scans a string finding its tokens
type lexer struct {
	input  string  // string being tokenized
	pos    int     // current position in the input string
	width  int     // width of last rune read from input
	start  int     // start position of the current token
	tokens []Token // tokens extracted from the input
}

func GetTokens(s string) []Token {
	const (
		whitespace   = " \t"
		upperLetters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		lowerLetters = "abcdefghijklmnopqrstuvwxyz"
		letters      = upperLetters + lowerLetters
		digits       = "0123456789"
		alphanumeric = letters + digits
		literalChar  = "'"
		timeChars    = literalChar + "-:."
		symbolChars  = alphanumeric + timeChars
	)

	l := &lexer{input: s}
	for {
		switch r := l.next(); { // Consume next rune
		case r == eof:
			l.emit(TokenEOF)
			return l.tokens

		case strings.ContainsRune(whitespace, r): // Skip any whitespace
			l.acceptRun(whitespace)
			l.emit(TokenWhitespace)

		case r == '(':
			l.emit(TokenLeftParen)

		case r == ')':
			l.emit(TokenRightParen)

		case r == ',':
			l.emit(TokenComma)

		case strings.ContainsRune("+-"+digits, r): // Number/Float if starts with +, -, or a digit
			l.acceptOne("+-")
			l.acceptRun(digits)
			if l.acceptOne(".") { // Decimal point?
				l.acceptRun(digits)
			}
			l.emit(TokenNumber)

		case strings.ContainsRune(alphanumeric+"'", r): // Symbol if starts with a letter
			l.acceptRun(symbolChars)
			l.emit(TokenSymbol)

		default:
			l.tokens = append(l.tokens,
				Token{TokenKind: TokenError, Symbol: fmt.Sprintf("Invalid character: %s", l.input[l.start:])})
			return l.tokens
		}
	}
}

// next reads the next rune or eof
func (l *lexer) next() (rune rune) {
	if l.pos >= len(l.input) { // At end of string
		l.width = 0
		return eof
	}
	rune, l.width = utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += l.width
	return rune
}

// backup places the previously read rune back
func (l *lexer) backup() { l.pos -= l.width }

func (l *lexer) acceptOne(validChars string) bool {
	if strings.ContainsRune(validChars, l.next()) {
		return true
	}
	l.backup()
	return false
}

func (l *lexer) acceptRun(valid string) {
	for strings.ContainsRune(valid, l.next()) {
	}
	l.backup()
}

func (l *lexer) emit(tk TokenKind) {
	t := Token{TokenKind: tk, Symbol: l.input[l.start:l.pos]}
	showToken(t)
	if tk != TokenWhitespace { // If whitespace, do NOT emit this token
		l.tokens = append(l.tokens, t)
	}
	l.start = l.pos
}

// eof represents a marker rune for the end of the reader.
const eof = rune(0)
