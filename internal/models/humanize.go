package models

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func HumanizeConcept(name string) string {
	words := strings.FieldsFunc(name, func(r rune) bool { return r == '-' || r == '_' })
	if len(words) == 0 {
		return name
	}
	text := strings.Join(words, " ")
	first, size := utf8.DecodeRuneInString(text)
	return string(unicode.ToUpper(first)) + text[size:]
}
