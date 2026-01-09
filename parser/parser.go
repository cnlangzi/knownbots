package parser

import (
	"io"
	"net/netip"
)

type Parser interface {
	Parse(r io.Reader) ([]netip.Prefix, error)
	Name() string
}

var parserRegistry = make(map[string]Parser)

func RegisterParser(name string, p Parser) {
	parserRegistry[name] = p
}

func Get(name string) Parser {
	if p := parserRegistry[name]; p != nil {
		return p
	}
	return parserRegistry["txt"]
}
