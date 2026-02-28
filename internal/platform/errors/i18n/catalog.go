// Package i18n provides internationalization support for error messages.
package i18n

import (
	"bytes"
	"strings"
	"sync"
	"text/template"

	i18ncatalog "github.com/louisbranch/fracturing.space/internal/platform/i18n/catalog"
)

// Code is a machine-readable error code (duplicated from errors package to avoid cycle).
type Code = string

// Catalog maps error codes to message templates for a specific locale.
type Catalog struct {
	locale   string
	messages map[Code]string
}

var (
	catalogsMu sync.RWMutex
	// catalogs holds override and runtime-built catalogs by locale.
	catalogs = map[string]*Catalog{}
)

// GetCatalog returns the catalog for the given locale.
// Falls back to en-US if the locale is not found.
func GetCatalog(locale string) *Catalog {
	requested := strings.TrimSpace(locale)
	if requested == "" {
		requested = i18ncatalog.BaseLocale
	}

	if c, ok := lookupCatalog(requested); ok {
		return c
	}

	resolvedLocale, messages := i18ncatalog.Default().NamespaceMessagesWithFallback(requested, "errors")
	if c, ok := lookupCatalog(resolvedLocale); ok {
		return c
	}

	built := NewCatalog(resolvedLocale, toCodeMap(messages))
	return storeCatalogIfAbsent(resolvedLocale, built)
}

// Locale returns the locale of this catalog.
func (c *Catalog) Locale() string {
	return c.locale
}

// Format renders the message template with the given metadata.
// Falls back to the error code itself if no template is found.
// Templates are always executed even with nil/empty metadata to ensure
// consistent output (template variables without metadata render as empty).
func (c *Catalog) Format(code Code, metadata map[string]string) string {
	tmpl, ok := c.messages[code]
	if !ok {
		return code
	}

	// Ensure metadata is non-nil for template execution
	if metadata == nil {
		metadata = map[string]string{}
	}

	// Parse and execute the template
	t, err := template.New("msg").Parse(tmpl)
	if err != nil {
		return tmpl
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, metadata); err != nil {
		return tmpl
	}
	return buf.String()
}

// RegisterCatalog registers a new catalog for the given locale.
// This is primarily for testing purposes. Callers should only use this
// during init or in single-threaded test setup, as the catalogs map
// is not protected by synchronization.
func RegisterCatalog(locale string, cat *Catalog) {
	catalogsMu.Lock()
	defer catalogsMu.Unlock()
	catalogs[locale] = cat
}

// NewCatalog creates a new catalog with the given locale and messages.
func NewCatalog(locale string, messages map[Code]string) *Catalog {
	cloned := make(map[Code]string, len(messages))
	for key, value := range messages {
		cloned[key] = value
	}
	return &Catalog{
		locale:   locale,
		messages: cloned,
	}
}

func lookupCatalog(locale string) (*Catalog, bool) {
	catalogsMu.RLock()
	defer catalogsMu.RUnlock()
	cat, ok := catalogs[locale]
	return cat, ok
}

func storeCatalogIfAbsent(locale string, candidate *Catalog) *Catalog {
	catalogsMu.Lock()
	defer catalogsMu.Unlock()
	if existing, ok := catalogs[locale]; ok {
		return existing
	}
	catalogs[locale] = candidate
	return candidate
}

func toCodeMap(messages map[string]string) map[Code]string {
	out := make(map[Code]string, len(messages))
	for key, value := range messages {
		out[Code(key)] = value
	}
	return out
}
