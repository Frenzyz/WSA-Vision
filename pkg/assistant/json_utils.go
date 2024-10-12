package assistant

import (
	"fmt"
	"regexp"
	"strings"
)

// ExtractJSON attempts to find and return the first JSON object or array in the input string.
func ExtractJSON(input string) (string, error) {
	// Remove any leading or trailing code fences or backticks
	input = strings.TrimSpace(input)
	input = strings.Trim(input, "```")
	input = strings.TrimPrefix(input, "json")
	input = strings.TrimSpace(input)

	// Regular expression to match JSON objects or arrays
	jsonRegex := regexp.MustCompile(`(?s)\{.*\}|\[.*\]`)
	matches := jsonRegex.FindAllString(input, -1)
	if len(matches) == 0 {
		return "", fmt.Errorf("no JSON object or array found in the input")
	}
	// Return the first match
	return matches[0], nil
}
