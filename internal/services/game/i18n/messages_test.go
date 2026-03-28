package i18n

import (
	"testing"

	platformcatalog "github.com/louisbranch/fracturing.space/internal/platform/i18n/catalog"
)

func TestExportedGameMessagesExistInAllCatalogLocales(t *testing.T) {
	bundle := platformcatalog.Default()
	if bundle == nil {
		t.Fatal("expected embedded catalog bundle")
	}

	contracts := Contracts()
	if len(contracts) == 0 {
		t.Fatal("expected game i18n contracts")
	}

	locales := bundle.Locales()
	if len(locales) == 0 {
		t.Fatal("expected embedded catalog locales")
	}

	for _, contract := range contracts {
		contract := contract
		t.Run(contract.Key, func(t *testing.T) {
			if contract.Key == "" {
				t.Fatal("message contract key should not be blank")
			}
			if contract.Fallback == "" {
				t.Fatalf("message contract %q fallback should not be blank", contract.Key)
			}

			for _, locale := range locales {
				locale := locale
				t.Run(locale, func(t *testing.T) {
					messages := bundle.LocaleMessages(locale)
					value, ok := messages[contract.Key]
					if !ok {
						t.Fatalf("catalog locale %q is missing key %q", locale, contract.Key)
					}
					if value == "" {
						t.Fatalf("catalog locale %q key %q should not be blank", locale, contract.Key)
					}
				})
			}

			baseValue, ok := bundle.Message(platformcatalog.BaseLocale, contract.Key)
			if !ok {
				t.Fatalf("base locale %q is missing key %q", platformcatalog.BaseLocale, contract.Key)
			}
			if baseValue != contract.Fallback {
				t.Fatalf("base locale copy for %q = %q, want fallback %q", contract.Key, baseValue, contract.Fallback)
			}
		})
	}
}
