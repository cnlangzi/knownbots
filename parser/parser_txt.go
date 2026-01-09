package parser

import (
	"bufio"
	"io"
	"net/netip"
	"strings"
)

type TxtParser struct{}

func (p *TxtParser) Name() string {
	return "txt"
}

func (p *TxtParser) Parse(r io.Reader) ([]netip.Prefix, error) {
	var prefixes []netip.Prefix
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		prefix, err := netip.ParsePrefix(line)
		if err == nil {
			prefixes = append(prefixes, prefix)
		}
	}
	return prefixes, scanner.Err()
}

func init() {
	RegisterParser("txt", &TxtParser{})
}
