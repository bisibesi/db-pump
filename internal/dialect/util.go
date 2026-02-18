package dialect

import (
	"strings"
)

// GeneratePlaceholders is a helper function to create a slice of placeholder strings.
// It takes the number of placeholders needed and a function that returns the placeholder for a given index.
// It returns a comma-separated string of the generated placeholders.
func GeneratePlaceholders(count int, placeholderFunc func(int) string) string {
	placeholders := make([]string, count)
	for i := 0; i < count; i++ {
		placeholders[i] = placeholderFunc(i)
	}
	return strings.Join(placeholders, ", ")
}

// DefaultNormalizeType is a default implementation for type normalization (lowercase).
func DefaultNormalizeType(sqlType string) string {
	return strings.ToLower(sqlType)
}

// DefaultGetSchemaName is a default implementation for Getting Schema Name (identity).
func DefaultGetSchemaName(input string) string {
	return input
}
