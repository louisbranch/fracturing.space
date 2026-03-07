// Package main renders translator-friendly i18n status artifacts.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	i18ncatalog "github.com/louisbranch/fracturing.space/internal/platform/i18n/catalog"
)

type report struct {
	BaseLocale string         `json:"base_locale"`
	Locales    []localeStatus `json:"locales"`
}

type localeStatus struct {
	Locale      string            `json:"locale"`
	BaseKeys    int               `json:"base_keys"`
	Translated  int               `json:"translated"`
	Missing     int               `json:"missing"`
	Extra       int               `json:"extra"`
	Completion  float64           `json:"completion"`
	Namespaces  []namespaceStatus `json:"namespaces"`
	MissingKeys []string          `json:"missing_keys"`
	ExtraKeys   []string          `json:"extra_keys"`
}

type namespaceStatus struct {
	Namespace  string  `json:"namespace"`
	BaseKeys   int     `json:"base_keys"`
	Translated int     `json:"translated"`
	Missing    int     `json:"missing"`
	Extra      int     `json:"extra"`
	Completion float64 `json:"completion"`
}

type codedError struct {
	code int
	err  error
}

func (e codedError) Error() string {
	return e.err.Error()
}

func (e codedError) Unwrap() error {
	return e.err
}

func withExitCode(err error, code int) error {
	if err == nil {
		return nil
	}
	return codedError{code: code, err: err}
}

func exitCode(err error) int {
	var codeErr codedError
	if errors.As(err, &codeErr) {
		return codeErr.code
	}
	return 1
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		os.Exit(exitCode(err))
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	var baseLocale string
	var markdownOut string
	var jsonOut string

	flags := flag.NewFlagSet("i18nstatus", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&baseLocale, "base-locale", i18ncatalog.BaseLocale, "base locale used as translation source of truth")
	flags.StringVar(&markdownOut, "out", "docs/reference/i18n-status.md", "markdown output path")
	flags.StringVar(&jsonOut, "json-out", "docs/reference/i18n-status.json", "json output path")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return withExitCode(err, 2)
	}

	bundle, err := i18ncatalog.LoadEmbedded()
	if err != nil {
		fmt.Fprintf(stderr, "load i18n catalogs: %v\n", err)
		return withExitCode(err, 1)
	}
	if !bundle.HasLocale(baseLocale) {
		err := fmt.Errorf("base locale %q is missing from catalogs", baseLocale)
		fmt.Fprintf(stderr, "%v\n", err)
		return withExitCode(err, 1)
	}

	rep := buildReport(bundle, baseLocale)
	if err := writeJSON(jsonOut, rep); err != nil {
		fmt.Fprintf(stderr, "write json report: %v\n", err)
		return withExitCode(err, 1)
	}
	if err := writeMarkdown(markdownOut, rep); err != nil {
		fmt.Fprintf(stderr, "write markdown report: %v\n", err)
		return withExitCode(err, 1)
	}
	fmt.Fprintf(stdout, "wrote %s and %s\n", markdownOut, jsonOut)
	return nil
}

func buildReport(bundle *i18ncatalog.Bundle, baseLocale string) report {
	baseMessages := bundle.LocaleMessages(baseLocale)
	baseNamespaces := bundle.Namespaces(baseLocale)
	baseNamespaceSet := map[string]struct{}{}
	for _, namespace := range baseNamespaces {
		baseNamespaceSet[namespace] = struct{}{}
	}

	locales := bundle.Locales()
	statuses := make([]localeStatus, 0, len(locales))
	for _, locale := range locales {
		localeMessages := bundle.LocaleMessages(locale)
		missingKeyList := missingKeys(baseMessages, localeMessages)
		extraKeyList := extraKeys(baseMessages, localeMessages)
		translated := len(baseMessages) - len(missingKeyList)
		completion := percent(translated, len(baseMessages))

		namespaceUnionSet := map[string]struct{}{}
		for namespace := range baseNamespaceSet {
			namespaceUnionSet[namespace] = struct{}{}
		}
		for _, namespace := range bundle.Namespaces(locale) {
			namespaceUnionSet[namespace] = struct{}{}
		}
		namespaceUnion := sortedSetKeys(namespaceUnionSet)

		namespaceStatuses := make([]namespaceStatus, 0, len(namespaceUnion))
		for _, namespace := range namespaceUnion {
			baseNS := bundle.NamespaceMessages(baseLocale, namespace)
			localeNS := bundle.NamespaceMessages(locale, namespace)
			nsMissing := missingKeys(baseNS, localeNS)
			nsExtra := extraKeys(baseNS, localeNS)
			nsTranslated := len(baseNS) - len(nsMissing)
			namespaceStatuses = append(namespaceStatuses, namespaceStatus{
				Namespace:  namespace,
				BaseKeys:   len(baseNS),
				Translated: nsTranslated,
				Missing:    len(nsMissing),
				Extra:      len(nsExtra),
				Completion: percent(nsTranslated, len(baseNS)),
			})
		}

		statuses = append(statuses, localeStatus{
			Locale:      locale,
			BaseKeys:    len(baseMessages),
			Translated:  translated,
			Missing:     len(missingKeyList),
			Extra:       len(extraKeyList),
			Completion:  completion,
			Namespaces:  namespaceStatuses,
			MissingKeys: missingKeyList,
			ExtraKeys:   extraKeyList,
		})
	}

	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].Locale < statuses[j].Locale
	})

	return report{BaseLocale: baseLocale, Locales: statuses}
}

func writeJSON(path string, rep report) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}
	data, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func writeMarkdown(path string, rep report) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}

	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("title: \"I18n status\"\n")
	b.WriteString("parent: \"Reference\"\n")
	b.WriteString("nav_order: 20\n")
	b.WriteString("---\n\n")
	b.WriteString("# I18n Status\n\n")
	b.WriteString("Generated by `make i18n-status`.\n\n")
	b.WriteString("Base locale: `")
	b.WriteString(rep.BaseLocale)
	b.WriteString("`.\n\n")

	b.WriteString("## Locale Summary\n\n")
	b.WriteString("| Locale | Base Keys | Translated | Missing | Extra | Completion |\n")
	b.WriteString("| --- | ---: | ---: | ---: | ---: | ---: |\n")
	for _, locale := range rep.Locales {
		b.WriteString(fmt.Sprintf("| `%s` | %d | %d | %d | %d | %.1f%% |\n", locale.Locale, locale.BaseKeys, locale.Translated, locale.Missing, locale.Extra, locale.Completion))
	}

	for _, locale := range rep.Locales {
		b.WriteString("\n## Locale: `")
		b.WriteString(locale.Locale)
		b.WriteString("`\n\n")

		b.WriteString("### Namespace Summary\n\n")
		b.WriteString("| Namespace | Base Keys | Translated | Missing | Extra | Completion |\n")
		b.WriteString("| --- | ---: | ---: | ---: | ---: | ---: |\n")
		for _, ns := range locale.Namespaces {
			b.WriteString(fmt.Sprintf("| `%s` | %d | %d | %d | %d | %.1f%% |\n", ns.Namespace, ns.BaseKeys, ns.Translated, ns.Missing, ns.Extra, ns.Completion))
		}

		if len(locale.MissingKeys) > 0 {
			b.WriteString("\n### Missing Keys\n\n")
			for _, key := range locale.MissingKeys {
				b.WriteString("- `")
				b.WriteString(key)
				b.WriteString("`\n")
			}
		}
		if len(locale.ExtraKeys) > 0 {
			b.WriteString("\n### Extra Keys\n\n")
			for _, key := range locale.ExtraKeys {
				b.WriteString("- `")
				b.WriteString(key)
				b.WriteString("`\n")
			}
		}
	}

	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func missingKeys(base map[string]string, target map[string]string) []string {
	out := make([]string, 0)
	for key := range base {
		if _, ok := target[key]; !ok {
			out = append(out, key)
		}
	}
	sort.Strings(out)
	return out
}

func extraKeys(base map[string]string, target map[string]string) []string {
	out := make([]string, 0)
	for key := range target {
		if _, ok := base[key]; !ok {
			out = append(out, key)
		}
	}
	sort.Strings(out)
	return out
}

func sortedSetKeys(entries map[string]struct{}) []string {
	out := make([]string, 0, len(entries))
	for key := range entries {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func percent(numerator int, denominator int) float64 {
	if denominator <= 0 {
		return 100
	}
	value := float64(numerator) * 100 / float64(denominator)
	return math.Round(value*10) / 10
}
