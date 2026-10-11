package dot

import (
	"fmt"
	"strings"
)

func Plural(n int, what string) string {
	switch {
	case n == 1:
		return fmt.Sprintf("%d %s", n, what)
	case strings.HasSuffix(what, "s"):
		return fmt.Sprintf("%d %ses", n, what)
	}
	return fmt.Sprintf("%d %ss", n, what)
}
