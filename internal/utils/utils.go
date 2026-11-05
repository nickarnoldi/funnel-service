package utils

import (
	"net/http"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func ParseBoolean(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "1" || value == "true" || value == "yes" || value == "t" || value == "y" {
		return true
	}
	return false
}

// GetRequestIPAddress extracts the client's IP address from the HTTP request.
// It checks the "X-Forwarded-For" header first, which is used with proxies,
func GetRequestIPAddress(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return strings.Split(forwarded, ",")[0]
	}

	return r.RemoteAddr
}

func RemoveAllWhitespace(input string) string {
	return strings.ReplaceAll(input, " ", "")
}

func ToUppercaseWords(input string, langOpt ...language.Tag) string {
	lang := language.Und // Und is for undetermined language, and is a generally safe default
	if len(langOpt) > 0 {
		lang = langOpt[0]
	}

	caser := cases.Title(lang)
	return caser.String(input)
}
