package catalog

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

const (
	// BaseLocale is the canonical source locale for catalogs.
	BaseLocale = "en-US"
)

type catalogFile struct {
	Locale    string
	Namespace string
	Messages  map[string]string
}

// LocaleCatalog stores all messages for one locale, grouped by namespace.
type LocaleCatalog struct {
	Locale     string
	Namespaces map[string]map[string]string
	Messages   map[string]string
}

// Bundle contains all locale catalogs loaded from disk.
type Bundle struct {
	locales map[string]*LocaleCatalog
}

//go:embed locales/*/*.yaml
var embeddedCatalogFS embed.FS

var defaultBundle = mustLoadAndRegisterEmbedded()

// Default returns the process-wide embedded catalog bundle.
func Default() *Bundle {
	return defaultBundle
}

// LoadEmbedded loads catalog files embedded in this package.
func LoadEmbedded() (*Bundle, error) {
	return LoadFromFS(embeddedCatalogFS)
}

// LoadFromFS loads catalog files from the provided filesystem.
func LoadFromFS(catalogFS fs.FS) (*Bundle, error) {
	paths, err := fs.Glob(catalogFS, "locales/*/*.yaml")
	if err != nil {
		return nil, fmt.Errorf("glob locale catalogs: %w", err)
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no catalog files found")
	}
	sort.Strings(paths)

	bundle := &Bundle{locales: map[string]*LocaleCatalog{}}

	for _, path := range paths {
		data, err := fs.ReadFile(catalogFS, path)
		if err != nil {
			return nil, fmt.Errorf("read catalog %s: %w", path, err)
		}
		parsed, err := parseCatalogFile(data)
		if err != nil {
			return nil, fmt.Errorf("parse catalog %s: %w", path, err)
		}
		if err := bundle.addFile(path, parsed); err != nil {
			return nil, err
		}
	}

	if !bundle.HasLocale(BaseLocale) {
		return nil, fmt.Errorf("base locale %s is not defined in catalogs", BaseLocale)
	}

	return bundle, nil
}

func (b *Bundle) addFile(path string, file catalogFile) error {
	localeFromPath := filepath.Base(filepath.Dir(path))
	namespaceFromPath := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))

	locale := strings.TrimSpace(file.Locale)
	if locale == "" {
		return fmt.Errorf("catalog %s: locale is required", path)
	}
	if locale != localeFromPath {
		return fmt.Errorf("catalog %s: locale %q must match path locale %q", path, locale, localeFromPath)
	}

	namespace := strings.TrimSpace(file.Namespace)
	if namespace == "" {
		return fmt.Errorf("catalog %s: namespace is required", path)
	}
	if namespace != namespaceFromPath {
		return fmt.Errorf("catalog %s: namespace %q must match filename namespace %q", path, namespace, namespaceFromPath)
	}

	if file.Messages == nil {
		return fmt.Errorf("catalog %s: messages map is required", path)
	}

	localeCatalog, ok := b.locales[locale]
	if !ok {
		localeCatalog = &LocaleCatalog{
			Locale:     locale,
			Namespaces: map[string]map[string]string{},
			Messages:   map[string]string{},
		}
		b.locales[locale] = localeCatalog
	}
	if _, exists := localeCatalog.Namespaces[namespace]; exists {
		return fmt.Errorf("catalog %s: namespace %q already defined for locale %q", path, namespace, locale)
	}

	namespaceMessages := make(map[string]string, len(file.Messages))
	for key, value := range file.Messages {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			return fmt.Errorf("catalog %s: message key cannot be blank", path)
		}
		if strings.HasPrefix(trimmedKey, "core.") && namespace != "core" {
			return fmt.Errorf("catalog %s: key %q must be defined in core namespace", path, trimmedKey)
		}
		if _, exists := localeCatalog.Messages[trimmedKey]; exists {
			return fmt.Errorf("catalog %s: duplicate key %q in locale %q", path, trimmedKey, locale)
		}

		localeCatalog.Messages[trimmedKey] = value
		namespaceMessages[trimmedKey] = value
	}

	localeCatalog.Namespaces[namespace] = namespaceMessages
	return nil
}

// Register registers all catalog messages with x/text/message.
func (b *Bundle) Register() error {
	if b == nil {
		return nil
	}
	locales := b.Locales()
	for _, locale := range locales {
		tag, err := language.Parse(locale)
		if err != nil {
			return fmt.Errorf("parse locale tag %q: %w", locale, err)
		}
		tags := []language.Tag{tag}
		if base, _ := tag.Base(); base.String() != "" && base.String() != "und" {
			baseTag, err := language.Parse(base.String())
			if err == nil && baseTag.String() != tag.String() {
				tags = append(tags, baseTag)
			}
		}
		messages := b.LocaleMessages(locale)
		keys := make([]string, 0, len(messages))
		for key := range messages {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			for _, registerTag := range tags {
				message.SetString(registerTag, key, messages[key])
			}
		}
	}
	return nil
}

// HasLocale reports whether the locale exists in this bundle.
func (b *Bundle) HasLocale(locale string) bool {
	if b == nil {
		return false
	}
	_, ok := b.locales[strings.TrimSpace(locale)]
	return ok
}

// Locales returns all available locale identifiers.
func (b *Bundle) Locales() []string {
	if b == nil {
		return nil
	}
	out := make([]string, 0, len(b.locales))
	for locale := range b.locales {
		out = append(out, locale)
	}
	sort.Strings(out)
	return out
}

// LocaleMessages returns an exact locale message map copy.
func (b *Bundle) LocaleMessages(locale string) map[string]string {
	if b == nil {
		return map[string]string{}
	}
	catalog, ok := b.locales[strings.TrimSpace(locale)]
	if !ok || catalog == nil {
		return map[string]string{}
	}
	return copyMap(catalog.Messages)
}

// Messages returns a locale message map with base-locale fallback.
func (b *Bundle) Messages(locale string) map[string]string {
	if messages := b.LocaleMessages(locale); len(messages) > 0 {
		return messages
	}
	return b.LocaleMessages(BaseLocale)
}

// Message returns one message value with base-locale fallback.
func (b *Bundle) Message(locale string, key string) (string, bool) {
	if b == nil {
		return "", false
	}
	trimmedLocale := strings.TrimSpace(locale)
	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" {
		return "", false
	}
	if catalog, ok := b.locales[trimmedLocale]; ok && catalog != nil {
		if value, exists := catalog.Messages[trimmedKey]; exists {
			return value, true
		}
	}
	if trimmedLocale != BaseLocale {
		if catalog, ok := b.locales[BaseLocale]; ok && catalog != nil {
			value, exists := catalog.Messages[trimmedKey]
			return value, exists
		}
	}
	return "", false
}

// Namespaces returns sorted namespace names for a locale.
func (b *Bundle) Namespaces(locale string) []string {
	if b == nil {
		return nil
	}
	catalog, ok := b.locales[strings.TrimSpace(locale)]
	if !ok || catalog == nil {
		return nil
	}
	out := make([]string, 0, len(catalog.Namespaces))
	for namespace := range catalog.Namespaces {
		out = append(out, namespace)
	}
	sort.Strings(out)
	return out
}

// NamespaceMessages returns an exact namespace message map copy for a locale.
func (b *Bundle) NamespaceMessages(locale string, namespace string) map[string]string {
	if b == nil {
		return map[string]string{}
	}
	catalog, ok := b.locales[strings.TrimSpace(locale)]
	if !ok || catalog == nil {
		return map[string]string{}
	}
	messages, ok := catalog.Namespaces[strings.TrimSpace(namespace)]
	if !ok {
		return map[string]string{}
	}
	return copyMap(messages)
}

// NamespaceMessagesWithFallback returns namespace messages and the locale that satisfied the lookup.
func (b *Bundle) NamespaceMessagesWithFallback(locale string, namespace string) (string, map[string]string) {
	trimmedLocale := strings.TrimSpace(locale)
	trimmedNamespace := strings.TrimSpace(namespace)
	if messages := b.NamespaceMessages(trimmedLocale, trimmedNamespace); len(messages) > 0 {
		return trimmedLocale, messages
	}
	return BaseLocale, b.NamespaceMessages(BaseLocale, trimmedNamespace)
}

func copyMap(source map[string]string) map[string]string {
	out := make(map[string]string, len(source))
	for key, value := range source {
		out[key] = value
	}
	return out
}

func mustLoadAndRegisterEmbedded() *Bundle {
	bundle, err := LoadEmbedded()
	if err != nil {
		panic(err)
	}
	if err := bundle.Register(); err != nil {
		panic(err)
	}
	return bundle
}

func parseCatalogFile(data []byte) (catalogFile, error) {
	lines := strings.Split(string(data), "\n")
	out := catalogFile{Messages: map[string]string{}}
	state := ""

	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		switch {
		case strings.HasPrefix(line, "locale:"):
			value, err := parseQuotedValue(strings.TrimSpace(strings.TrimPrefix(line, "locale:")))
			if err != nil {
				return catalogFile{}, fmt.Errorf("parse locale: %w", err)
			}
			out.Locale = value
		case strings.HasPrefix(line, "namespace:"):
			value, err := parseQuotedValue(strings.TrimSpace(strings.TrimPrefix(line, "namespace:")))
			if err != nil {
				return catalogFile{}, fmt.Errorf("parse namespace: %w", err)
			}
			out.Namespace = value
		case line == "messages:":
			state = "messages"
		default:
			if state != "messages" {
				return catalogFile{}, fmt.Errorf("unexpected line %q", line)
			}
			key, value, err := parseMessageEntry(line)
			if err != nil {
				return catalogFile{}, fmt.Errorf("parse message entry %q: %w", line, err)
			}
			out.Messages[key] = value
		}
	}

	if out.Locale == "" {
		return catalogFile{}, fmt.Errorf("missing locale")
	}
	if out.Namespace == "" {
		return catalogFile{}, fmt.Errorf("missing namespace")
	}
	if len(out.Messages) == 0 {
		return catalogFile{}, fmt.Errorf("missing messages")
	}

	return out, nil
}

func parseMessageEntry(line string) (string, string, error) {
	keyToken, rest, err := splitQuotedToken(line)
	if err != nil {
		return "", "", err
	}
	key, err := strconv.Unquote(keyToken)
	if err != nil {
		return "", "", fmt.Errorf("unquote key: %w", err)
	}

	rest = strings.TrimSpace(rest)
	if !strings.HasPrefix(rest, ":") {
		return "", "", fmt.Errorf("missing ':' separator")
	}
	valueToken := strings.TrimSpace(strings.TrimPrefix(rest, ":"))
	value, err := parseQuotedValue(valueToken)
	if err != nil {
		return "", "", fmt.Errorf("unquote value: %w", err)
	}
	return key, value, nil
}

func parseQuotedValue(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	parsed, err := strconv.Unquote(trimmed)
	if err != nil {
		return "", err
	}
	return parsed, nil
}

func splitQuotedToken(line string) (string, string, error) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "\"") {
		return "", "", fmt.Errorf("expected quoted token")
	}
	escaped := false
	for i := 1; i < len(trimmed); i++ {
		ch := trimmed[i]
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		if ch == '"' {
			return trimmed[:i+1], trimmed[i+1:], nil
		}
	}
	return "", "", fmt.Errorf("unterminated quoted token")
}
