package asn

import (
	"testing"
)

func TestBGPHERegex(t *testing.T) {
	tests := []struct {
		name string
		html string
		want []string
	}{
		{
			name: "IPv4 prefix",
			html: `<a href="/net/31.13.24.0/21">31.13.24.0/21</a>`,
			want: []string{"31.13.24.0/21"},
		},
		{
			name: "IPv6 prefix",
			html: `<a href="/net/2001:db8::/32">2001:db8::/32</a>`,
			want: []string{"2001:db8::/32"},
		},
		{
			name: "IPv6 with lowercase",
			html: `<a href="/net/2a03:2880:f003::/48">2a03:2880:f003::/48</a>`,
			want: []string{"2a03:2880:f003::/48"},
		},
		{
			name: "IPv6 with uppercase",
			html: `<a href="/net/2A03:2880:F003::/48">2A03:2880:F003::/48</a>`,
			want: []string{"2A03:2880:F003::/48"},
		},
		{
			name: "mixed IPv4 and IPv6",
			html: `
				<a href="/net/31.13.24.0/21">31.13.24.0/21</a>
				<a href="/net/2001:db8::/32">2001:db8::/32</a>
				<a href="/net/157.240.0.0/16">157.240.0.0/16</a>
			`,
			want: []string{"31.13.24.0/21", "2001:db8::/32", "157.240.0.0/16"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := bgpHePrefixRe.FindAllStringSubmatch(tt.html, -1)

			got := make([]string, 0, len(matches))
			for _, match := range matches {
				if len(match) >= 2 {
					got = append(got, match[1])
				}
			}

			if len(got) != len(tt.want) {
				t.Errorf("got %d matches, want %d", len(got), len(tt.want))
				t.Errorf("got: %v", got)
				t.Errorf("want: %v", tt.want)
				return
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("match[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
