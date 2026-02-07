// Package i18n provides internationalization support for error messages.
package i18n

import (
	"bytes"
	"text/template"
)

// Code is a machine-readable error code (duplicated from errors package to avoid cycle).
type Code = string

// Catalog maps error codes to message templates for a specific locale.
type Catalog struct {
	locale   string
	messages map[Code]string
}

// catalogs holds all available message catalogs by locale.
var catalogs = map[string]*Catalog{
	"en-US": enUSCatalog,
}

// GetCatalog returns the catalog for the given locale.
// Falls back to en-US if the locale is not found.
func GetCatalog(locale string) *Catalog {
	if c, ok := catalogs[locale]; ok {
		return c
	}
	return catalogs["en-US"]
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
	catalogs[locale] = cat
}

// NewCatalog creates a new catalog with the given locale and messages.
func NewCatalog(locale string, messages map[Code]string) *Catalog {
	return &Catalog{
		locale:   locale,
		messages: messages,
	}
}
