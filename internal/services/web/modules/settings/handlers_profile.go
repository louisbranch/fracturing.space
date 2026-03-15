package settings

import (
	"net/http"

	settingsapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/settings/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	flashnotice "github.com/louisbranch/fracturing.space/internal/services/web/platform/flash"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/forminput"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// handleProfileGet handles this route in the module transport layer.
func (h handlers) handleProfileGet(w http.ResponseWriter, r *http.Request) {
	ctx, userID := h.RequestContextAndUserID(r)
	profile, err := h.account.LoadProfile(ctx, userID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	h.renderProfilePage(w, r, http.StatusOK, profile, "")
}

// handleProfilePost handles this route in the module transport layer.
func (h handlers) handleProfilePost(w http.ResponseWriter, r *http.Request) {
	ctx, userID := h.RequestContextAndUserID(r)
	if err := forminput.ParseInvalidInput(r, "error.web.message.failed_to_parse_profile_form", "failed to parse profile form"); err != nil {
		h.WriteError(w, r, err)
		return
	}
	existingProfile, err := h.account.LoadProfile(ctx, userID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	profile := parseProfileInput(r.PostForm, existingProfile)
	if err := h.account.SaveProfile(ctx, userID, profile); err != nil {
		if apperrors.HTTPStatus(err) == http.StatusBadRequest {
			loc, lang := h.PageLocalizer(w, r)
			h.renderProfilePage(w, r, http.StatusBadRequest, profile, webi18n.LocalizeError(loc, err, lang))
			return
		}
		h.WriteError(w, r, err)
		return
	}
	h.sync.ProfileSaved(ctx, userID)
	h.writeFlashNotice(w, r, flashnotice.NoticeSuccess("web.settings.user_profile.notice_saved"))
	httpx.WriteRedirect(w, r, routepath.AppSettingsProfile)
}

// renderProfilePage centralizes this web behavior in one helper seam.
func (h handlers) renderProfilePage(w http.ResponseWriter, r *http.Request, statusCode int, profile settingsapp.SettingsProfile, errorMessage string) {
	loc, _ := h.PageLocalizer(w, r)
	h.writeSettingsPage(
		w,
		r,
		loc,
		statusCode,
		routepath.AppSettingsProfile,
		webtemplates.T(loc, "web.settings.page_profile_title"),
		SettingsProfileFragment(SettingsProfileForm{
			Username:      profile.Username,
			Name:          profile.Name,
			AvatarSetID:   profile.AvatarSetID,
			AvatarAssetID: profile.AvatarAssetID,
			Pronouns:      profile.Pronouns,
			Bio:           profile.Bio,
			ErrorMessage:  errorMessage,
		}, loc),
	)
}
