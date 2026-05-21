package handlers

import (
	"strings"
	"unicode/utf8"
)

const (
	maxNameLength        = 100
	maxTicketTitleLength = 255
	maxDescriptionLength = 5000
	maxCommentLength     = 2000
)

func normalizeEmail(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeText(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func normalizeLongText(value string) string {
	return strings.TrimSpace(value)
}

func textLength(value string) int {
	return utf8.RuneCountInString(value)
}
