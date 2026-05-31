package texttokens

import (
	"regexp"
	"strings"
)

var alnumPattern = regexp.MustCompile(`[a-z0-9]+`)

// LowerAlnum returns lower-cased ASCII alphanumeric tokens from value.
func LowerAlnum(value string) []string {
	return alnumPattern.FindAllString(strings.ToLower(value), -1)
}
