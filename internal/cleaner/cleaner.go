package cleaner

import (
	"regexp"
	"strings"
)

type Cleaner interface {
	Clean(string) string
}

type RegexCleaner struct {
	reBlockComments *regexp.Regexp
	reLineComments  *regexp.Regexp
	reSpaces        *regexp.Regexp
	reEmptyLines    *regexp.Regexp
}

func NewCleaner() *RegexCleaner {
	return &RegexCleaner{
		// "/* ... */" note this regex does not support nesting
		// idk how to support nested comments with regex
		reBlockComments: regexp.MustCompile(`(?s)/\*.*?\*/`),

		// "//" comments to end of line
		reLineComments: regexp.MustCompile(`//.*`),

		// one or more spaces / tabs
		reSpaces: regexp.MustCompile(`[ \t]+`),

		// empty lines and lines with trailing spaces
		reEmptyLines: regexp.MustCompile(`(?m)^\s*\n`),
	}
}

func (c *RegexCleaner) Clean(input string) string {
	out := input

	// 1. Remove block comments
	out = c.reBlockComments.ReplaceAllString(out, "")

	// 2. Remove line comments
	out = c.reLineComments.ReplaceAllString(out, "")

	// 3. Trim spaces at start and end of lines
	lines := strings.Split(out, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		line = c.reSpaces.ReplaceAllString(line, " ")
		lines[i] = line
	}
	out = strings.Join(lines, "\n")

	// 4. Remove empty lines
	out = c.reEmptyLines.ReplaceAllString(out, "")

	return strings.TrimSpace(out)
}
