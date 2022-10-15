package main

import (
	"fmt"
	"regexp"
	"strings"
)

var LinkCounter map[string]int = make(map[string]int)

var HTMLTagRegex = regexp.MustCompile(`<[^>]*>`)
var duplicateHyphenRegex = regexp.MustCompile(`-+`)

func GetAnchorLink(title string) string {
	// Remove any leading or trailing whitespace
	title = strings.TrimSpace(title)

	// Convert to lowercase
	title = strings.ToLower(title)

	// Remove any non-word characters
	//title = strings.Trim(title, )
	title = strings.Map(
		func(r rune) rune {
			nonWord := "!@#$%^&*()+={}[]|\\:;'<>?,./\""
			if strings.IndexRune(nonWord, r) < 0 {
				return r
			}
			return '@'
		},
		title,
	)
	title = strings.ReplaceAll(title, "@", "")
	// (remove HTML tags)
	title = HTMLTagRegex.ReplaceAllString(title, "")

	// Replace spaces with hyphens
	title = strings.Replace(title, " ", "-", -1)

	// Remove any duplicate hyphens

	// Duplicate hyphens REGEX
	title = duplicateHyphenRegex.ReplaceAllString(title, "-")

	// Remove any leading or trailing hyphens
	title = strings.Trim(title, "-")

	// Add the counter
	LinkCounter[title]++

	if LinkCounter[title] > 1 {
		title = fmt.Sprintf("#%s-%d", title, LinkCounter[title]-1)
		return title
	}

	return "#" + title
}
