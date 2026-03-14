package render

import (
	"regexp"
	"strings"
)

var classAllowedRe = regexp.MustCompile(`[^a-zA-Z0-9\-_:/\[\]().%# ]`)

func sanitizeClasses(s string) string {
	s = strings.NewReplacer("\n", " ", "\r", " ", "\t", " ").Replace(s)
	return classAllowedRe.ReplaceAllString(s, "")
}

func mergeClasses(base, extra string) string {
	base = strings.TrimSpace(base)
	extra = strings.TrimSpace(sanitizeClasses(extra))
	if base == "" {
		return extra
	}
	if extra == "" {
		return base
	}
	return base + " " + extra
}
