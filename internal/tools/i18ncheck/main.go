// Package main validates shared i18n catalogs for consistency and safety.
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	i18ncatalog "github.com/louisbranch/fracturing.space/internal/platform/i18n/catalog"
)

func main() {
	var baseLocale string
	var strictMissing bool
	flag.StringVar(&baseLocale, "base-locale", i18ncatalog.BaseLocale, "base locale used as translation source of truth")
	flag.BoolVar(&strictMissing, "strict-missing", false, "fail when non-base locales are missing base keys")
	flag.Parse()

	bundle, err := i18ncatalog.LoadEmbedded()
	if err != nil {
		fatalf("load i18n catalogs: %v", err)
	}

	if !bundle.HasLocale(baseLocale) {
		fatalf("base locale %q is missing from catalogs", baseLocale)
	}

	failures := make([]string, 0, 32)
	warnings := make([]string, 0, 32)

	for _, tag := range platformi18n.SupportedTags() {
		locale := tag.String()
		if !bundle.HasLocale(locale) {
			failures = append(failures, fmt.Sprintf("supported locale %q is missing from catalogs", locale))
		}
	}

	baseMessages := bundle.LocaleMessages(baseLocale)
	baseKeys := sortedKeys(baseMessages)
	locales := bundle.Locales()
	for _, locale := range locales {
		if locale == baseLocale {
			continue
		}
		localeMessages := bundle.LocaleMessages(locale)
		missing := 0
		extra := 0
		for _, key := range baseKeys {
			baseValue := baseMessages[key]
			translatedValue, ok := localeMessages[key]
			if !ok {
				missing++
				if strictMissing {
					failures = append(failures, fmt.Sprintf("locale %s missing key %s", locale, key))
				}
				continue
			}
			if !equalTokenMultiset(printfTokens(baseValue), printfTokens(translatedValue)) {
				failures = append(failures, fmt.Sprintf("locale %s key %s has mismatched printf placeholders", locale, key))
			}
			if !equalTokenMultiset(templateTokens(baseValue), templateTokens(translatedValue)) {
				failures = append(failures, fmt.Sprintf("locale %s key %s has mismatched template placeholders", locale, key))
			}
		}
		for key := range localeMessages {
			if _, ok := baseMessages[key]; !ok {
				extra++
			}
		}
		warnings = append(warnings, fmt.Sprintf("locale %s: missing=%d extra=%d", locale, missing, extra))
	}

	for _, line := range warnings {
		fmt.Println(line)
	}
	if len(failures) > 0 {
		for _, line := range failures {
			fmt.Fprintf(os.Stderr, "i18n check failure: %s\n", line)
		}
		os.Exit(1)
	}
	fmt.Println("i18n catalog check passed")
}

func printfTokens(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	verbs := map[byte]struct{}{
		'b': {}, 'c': {}, 'd': {}, 'e': {}, 'E': {}, 'f': {}, 'F': {}, 'g': {}, 'G': {},
		'o': {}, 'O': {}, 'p': {}, 'q': {}, 's': {}, 't': {}, 'T': {}, 'U': {}, 'v': {},
		'x': {}, 'X': {},
	}
	out := make([]string, 0, 4)
	for i := 0; i < len(value); i++ {
		if value[i] != '%' {
			continue
		}
		if i+1 < len(value) && value[i+1] == '%' {
			i++
			continue
		}
		j := i + 1
		for j < len(value) {
			if _, ok := verbs[value[j]]; ok {
				out = append(out, value[i:j+1])
				i = j
				break
			}
			if value[j] == '%' {
				break
			}
			j++
		}
	}
	sort.Strings(out)
	return out
}

func templateTokens(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	out := make([]string, 0, 4)
	for {
		start := strings.Index(value, "{{")
		if start < 0 {
			break
		}
		value = value[start+2:]
		end := strings.Index(value, "}}")
		if end < 0 {
			break
		}
		token := strings.TrimSpace(value[:end])
		value = value[end+2:]
		if strings.HasPrefix(token, ".") {
			name := strings.TrimSpace(strings.TrimPrefix(token, "."))
			if name != "" {
				out = append(out, name)
			}
		}
	}
	sort.Strings(out)
	return out
}

func equalTokenMultiset(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func sortedKeys(entries map[string]string) []string {
	keys := make([]string, 0, len(entries))
	for key := range entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
