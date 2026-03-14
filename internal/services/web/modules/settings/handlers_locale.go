package settings

import (
	"net/http"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	flashnotice "github.com/louisbranch/fracturing.space/internal/services/web/platform/flash"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/forminput"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// handleLocaleGet handles this route in the module transport layer.
func (h handlers) handleLocaleGet(w http.ResponseWriter, r *http.Request) {
	ctx, userID := h.RequestContextAndUserID(r)
	locale, err := h.account.LoadLocale(ctx, userID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	h.renderLocalePage(w, r, http.StatusOK, locale, "")
}

// handleLocalePost handles this route in the module transport layer.
func (h handlers) handleLocalePost(w http.ResponseWriter, r *http.Request) {
	ctx, userID := h.RequestContextAndUserID(r)
	if err := forminput.ParseInvalidInput(r, "error.web.message.failed_to_parse_locale_form", "failed to parse locale form"); err != nil {
		h.WriteError(w, r, err)
		return
	}
	selectedLocale := parseLocaleInput(r.PostForm)
	if err := h.account.SaveLocale(ctx, userID, selectedLocale); err != nil {
		if apperrors.HTTPStatus(err) == http.StatusBadRequest {
			loc, lang := h.PageLocalizer(w, r)
			h.renderLocalePage(w, r, http.StatusBadRequest, selectedLocale, webi18n.LocalizeError(loc, err, lang))
			return
		}
		h.WriteError(w, r, err)
		return
	}
	h.writeFlashNotice(w, r, flashnotice.NoticeSuccess("web.settings.locale.notice_saved"))
	httpx.WriteRedirect(w, r, routepath.AppSettingsLocale)
}

// renderLocalePage centralizes this web behavior in one helper seam.
func (h handlers) renderLocalePage(w http.ResponseWriter, r *http.Request, statusCode int, selectedLocale string, errorMessage string) {
	loc, _ := h.PageLocalizer(w, r)
	h.writeSettingsPage(
		w,
		r,
		loc,
		statusCode,
		routepath.AppSettingsLocale,
		webtemplates.T(loc, "web.settings.page_locale_title"),
		SettingsLocaleFragment(SettingsLocaleForm{
			SelectedLocale: selectedLocale,
			ErrorMessage:   errorMessage,
		}, loc),
	)
}
