package support

import (
	"net/http"

	"golang.org/x/text/message"

	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/i18n"
)

// ResolveLocalizer returns the request locale and printer and updates the locale
// cookie when locale negotiation requires it.
func ResolveLocalizer(w http.ResponseWriter, r *http.Request) (*message.Printer, string) {
	tag, setCookie := webi18n.ResolveTag(r)
	if setCookie {
		webi18n.SetLanguageCookie(w, tag)
	}
	return webi18n.Printer(tag), tag.String()
}
