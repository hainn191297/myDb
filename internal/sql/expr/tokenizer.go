package expr

import (
	"strconv"
	"strings"
	"unicode"
)

// tokenType represents the type of token.
type tokenType int

const (
	tokenIdent    tokenType = iota // id, email, status
	tokenNumber                    // 123, 45.67
	tokenString                    // 'hello', "world"
	tokenOperator                  // =, <, >, !=, <=, >=
	tokenKeyword                   // AND, OR, NOT, IS, NULL
	tokenLParen                    // (
	tokenRParen                    // )
	tokenEOF
)

// token represents a lexical token.
type token struct {
	typ tokenType
	val string
	pos int // position in input
}

// tokenize converts a WHERE clause string into tokens.
func tokenize(input string) []token {
	var tokens []token
	i := 0

	for i < len(input) {
		// Skip whitespace
		if isWhitespace(rune(input[i])) {
			i++
			continue
		}

		// Numbers (integers and floats)
		if isDigit(rune(input[i])) {
			start := i
			for i < len(input) && (isDigit(rune(input[i])) || input[i] == '.') {
				i++
			}
			tokens = append(tokens, token{tokenNumber, input[start:i], start})
			continue
		}

		// Strings (single or double quoted)
		if input[i] == '\'' || input[i] == '"' {
			quote := input[i]
			start := i
			i++
			for i < len(input) && input[i] != quote {
				if input[i] == '\\' && i+1 < len(input) {
					i++ // Skip escaped character
				}
				i++
			}
			if i < len(input) {
				i++ // closing quote
			}
			tokens = append(tokens, token{tokenString, input[start:i], start})
			continue
		}

		// Parentheses
		if input[i] == '(' {
			tokens = append(tokens, token{tokenLParen, "(", i})
			i++
			continue
		}
		if input[i] == ')' {
			tokens = append(tokens, token{tokenRParen, ")", i})
			i++
			continue
		}

		// Operators (=, !=, <, <=, >, >=)
		if isOperatorChar(input[i]) {
			start := i
			op := string(input[i])
			i++

			// Check for two-character operators (!=, <=, >=)
			if i < len(input) && input[i] == '=' {
				op += "="
				i++
			}

			tokens = append(tokens, token{tokenOperator, op, start})
			continue
		}

		// Keywords and Identifiers
		if isAlpha(rune(input[i])) {
			start := i
			for i < len(input) && (isAlphaNum(rune(input[i])) || input[i] == '_') {
				i++
			}
			word := input[start:i]
			wordUpper := strings.ToUpper(word)

			if isKeyword(wordUpper) {
				tokens = append(tokens, token{tokenKeyword, wordUpper, start})
			} else {
				tokens = append(tokens, token{tokenIdent, word, start})
			}
			continue
		}

		// Unknown character, skip
		i++
	}

	tokens = append(tokens, token{tokenEOF, "", i})
	return tokens
}

// Helper functions

func isWhitespace(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}

func isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}

func isAlpha(ch rune) bool {
	return unicode.IsLetter(ch)
}

func isAlphaNum(ch rune) bool {
	return unicode.IsLetter(ch) || unicode.IsDigit(ch)
}

func isOperatorChar(ch byte) bool {
	return ch == '=' || ch == '!' || ch == '<' || ch == '>'
}

func isKeyword(word string) bool {
	keywords := map[string]bool{
		"AND":  true,
		"OR":   true,
		"NOT":  true,
		"IS":   true,
		"NULL": true,
	}
	return keywords[word]
}

// ParseInt attempts to parse a string as int64.
func ParseInt(s string) (int64, bool) {
	val, err := strconv.ParseInt(s, 10, 64)
	return val, err == nil
}

// ParseFloat attempts to parse a string as float64.
func ParseFloat(s string) (float64, bool) {
	val, err := strconv.ParseFloat(s, 64)
	return val, err == nil
}

// StripQuotes removes surrounding quotes from a string literal.
func StripQuotes(s string) string {
	if len(s) >= 2 && (s[0] == '\'' || s[0] == '"') && s[0] == s[len(s)-1] {
		return s[1 : len(s)-1]
	}
	return s
}
