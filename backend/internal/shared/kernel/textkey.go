package kernel

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// SortKey is the text as it is compared when things are listed alphabetically and when two names are
// told apart: lowercase and without accents, so "Álamo" sorts next to "alamo" and "azúcar" before "Bicarbonato".
func SortKey(s string) string {
	var b strings.Builder
	for _, r := range norm.NFD.String(strings.TrimSpace(s)) {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}
