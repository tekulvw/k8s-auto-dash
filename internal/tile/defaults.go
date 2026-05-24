package tile

import (
	"strings"
	"unicode"
)

// DeriveName turns a hostname into a human display name by taking the
// first label, replacing '-' and '_' with spaces, and title-casing.
func DeriveName(hostname string) string {
	slug := firstLabel(hostname)
	if slug == "" {
		return ""
	}
	words := strings.FieldsFunc(slug, func(r rune) bool {
		return r == '-' || r == '_'
	})
	for i, w := range words {
		if len(w) == 0 {
			continue
		}
		runes := []rune(w)
		runes[0] = unicode.ToUpper(runes[0])
		for j := 1; j < len(runes); j++ {
			runes[j] = unicode.ToLower(runes[j])
		}
		words[i] = string(runes)
	}
	return strings.Join(words, " ")
}

// DeriveIconSlug returns the lowercase first label of the hostname,
// suitable for lookup against the bundled dashboard-icons set.
func DeriveIconSlug(hostname string) string {
	return strings.ToLower(firstLabel(hostname))
}

// DeriveURL returns the default URL for a hostname (https scheme).
func DeriveURL(hostname string) string {
	if hostname == "" {
		return ""
	}
	return "https://" + hostname
}

func firstLabel(hostname string) string {
	if hostname == "" {
		return ""
	}
	if i := strings.Index(hostname, "."); i >= 0 {
		return hostname[:i]
	}
	return hostname
}
