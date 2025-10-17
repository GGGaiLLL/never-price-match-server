package product

import (
	"fmt"
	"strconv"
	"strings"
)

// parsePrice is a new name for the old cleanPrice function.
func parsePrice(s string) (float64, error) {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "au$", "")
	s = strings.ReplaceAll(s, "a$", "")
	s = strings.ReplaceAll(s, "from", "")
	s = strings.Map(func(r rune) rune {
		switch {
		case r >= '0' && r <= '9', r == '.', r == ',':
			return r
		default:
			return -1
		}
	}, s)
	s = strings.ReplaceAll(s, ",", "")
	if s == "" {
		return 0, fmt.Errorf("empty price")
	}
	return strconv.ParseFloat(strings.TrimSpace(s), 64)
}
