package config

import (
	"fmt"
	"strings"
	"unicode"
)

// SplitArgs splits a shell-like argument string.
// Supports single/double quotes and backslash escaping outside single quotes.
func SplitArgs(input string) ([]string, error) {
	args := []string{}
	var current strings.Builder
	inSingle := false
	inDouble := false
	escaped := false

	flush := func() {
		if current.Len() > 0 {
			args = append(args, current.String())
			current.Reset()
		}
	}

	for _, r := range input {
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}

		if r == '\\' && !inSingle {
			escaped = true
			continue
		}

		if r == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}

		if r == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}

		if unicode.IsSpace(r) && !inSingle && !inDouble {
			flush()
			continue
		}

		current.WriteRune(r)
	}

	if escaped {
		return nil, fmt.Errorf("unterminated escape")
	}
	if inSingle || inDouble {
		return nil, fmt.Errorf("unterminated quote")
	}

	flush()
	return args, nil
}
