package configparser

import (
	"io"
	"text/scanner"
)

type ConfigParser struct {
	scanner.Scanner
	parenDepth int
}

func NewConfigParser(r io.Reader) *ConfigParser {
	p := &ConfigParser{}
	p.Init(r)
	p.IsIdentRune = isIdentRune
	return p
}

func isIdentRune(ch rune, i int) bool {
	if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || ch == '*' {
		return true
	}
	if i > 0 && ch == '.' {
		return true
	}
	return false
}
