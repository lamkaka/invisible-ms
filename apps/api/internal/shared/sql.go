package shared

import (
	"strings"
)

// SplitSQLStatements splits SQL content by semicolons, respecting string literals
// (single and double quotes). Returns non-empty trimmed statements.
func SplitSQLStatements(content string) []string {
	var statements []string
	var current strings.Builder
	inString := false
	var stringChar byte

	for i := 0; i < len(content); i++ {
		c := content[i]

		// Handle string literals (single or double quotes)
		if (c == '\'' || c == '"') && (i == 0 || content[i-1] != '\\') {
			if !inString {
				inString = true
				stringChar = c
			} else if c == stringChar {
				inString = false
			}
		}

		// Skip semicolons inside string literals
		if c == ';' && !inString {
			stmt := strings.TrimSpace(current.String())
			if stmt != "" {
				statements = append(statements, stmt)
			}
			current.Reset()
		} else {
			current.WriteByte(c)
		}
	}

	// Catch any remaining statement without trailing semicolon
	stmt := strings.TrimSpace(current.String())
	if stmt != "" {
		statements = append(statements, stmt)
	}

	return statements
}
