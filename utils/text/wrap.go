package text

import (
	"strings"
)

func Wrap(str string, width int) string {
	var builder strings.Builder
	pos := 0
	for _, char := range str {
		if char == '\n' {
			pos = 0
		} else if pos > width {
			pos = 1
			builder.WriteRune('\n')
		}
		builder.WriteRune(char)
		pos++
	}

	return builder.String()
}
