package model

import (
	"strings"
)

// splitWords splits a string into words
func splitWords(s string) []string {
	return strings.Fields(s)
}