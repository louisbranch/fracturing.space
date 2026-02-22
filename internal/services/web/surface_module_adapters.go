package web

import (
	"log"
	"net/http"
	"strings"

	authmodule "github.com/louisbranch/fracturing.space/internal/services/web/module/auth"
	discoverymodule "github.com/louisbranch/fracturing.space/internal/services/web/module/discovery"
	invitesmodule "github.com/louisbranch/fracturing.space/internal/services/web/module/invites"
	notificationsmodule "github.com/louisbranch/fracturing.space/internal/services/web/module/notifications"
	profilemodule "github.com/louisbranch/fracturing.space/internal/services/web/module/profile"
	publicprofilemodule "github.com/louisbranch/fracturing.space/internal/services/web/module/publicprofile"
	settingsmodule "github.com/louisbranch/fracturing.space/internal/services/web/module/settings"
	routepath "github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

type profileModuleService struct {
	handler *handler
}

func newProfileModuleService(h *handler) profilemodule.Service {
	if h == nil {
		return nil
	}
	return profileModuleService{handler: h}
}

func (s profileModuleService) HandleProfile(w http.ResponseWriter, r *http.Request) {
	s.handler.handleAppProfile(w, r)
}

type settingsModuleService struct {
	handler *handler
}

func newSettingsModuleService(h *handler) settingsmodule.Service {
	if h == nil {
		return nil
	}
	return settingsModuleService{handler: h}
}

func (s settingsModuleService) HandleSettings(w http.ResponseWriter, r *http.Request) {
	s.handler.handleAppSettings(w, r)
}

func (s settingsModuleService) HandleSettingsSubroutes(w http.ResponseWriter, r *http.Request) {
	s.handler.handleAppSettingsRoutes(w, r)
}

type invitesModuleService struct {
	handler *handler
}

func newInvitesModuleService(h *handler) invitesmodule.Service {
	if h == nil {
		return nil
	}
	return invitesModuleService{handler: h}
}

func (s invitesModuleService) HandleInvites(w http.ResponseWriter, r *http.Request) {
	s.handler.handleAppInvites(w, r)
}

func (s invitesModuleService) HandleInviteClaim(w http.ResponseWriter, r *http.Request) {
	s.handler.handleAppInviteClaim(w, r)
}

type notificationsModuleService struct {
	handler *handler
}

func newNotificationsModuleService(h *handler) notificationsmodule.Service {
	if h == nil {
		return nil
	}
	return notificationsModuleService{handler: h}
}

func (s notificationsModuleService) HandleNotifications(w http.ResponseWriter, r *http.Request) {
	s.handler.handleAppNotifications(w, r)
}

func (s notificationsModuleService) HandleNotificationsSubroutes(w http.ResponseWriter, r *http.Request) {
	s.handler.handleAppNotificationsRoutes(w, r)
}

type publicProfileModuleService struct {
	handler *handler
}

func newPublicProfileModuleService(h *handler) publicprofilemodule.Service {
	if h == nil {
		return nil
	}
	return publicProfileModuleService{handler: h}
}

func (s publicProfileModuleService) HandlePublicProfile(w http.ResponseWriter, r *http.Request) {
	s.handler.handlePublicProfile(w, r)
}

type discoveryModuleService struct {
	handler *handler
}

func newDiscoveryModuleService(h *handler) discoverymodule.Service {
	if h == nil {
		return nil
	}
	return discoveryModuleService{handler: h}
}

func (s discoveryModuleService) HandleDiscover(w http.ResponseWriter, r *http.Request) {
	s.handler.handleDiscover(w, r)
}

func (s discoveryModuleService) HandleDiscoverCampaign(w http.ResponseWriter, r *http.Request) {
	s.handler.handleDiscoverCampaign(w, r)
}

type authPublicModuleService struct {
	handler *handler
	appName string
}

func newAuthPublicModuleService(h *handler, appName string) authmodule.PublicService {
	if h == nil {
		return nil
	}
	return authPublicModuleService{
		handler: h,
		appName: appName,
	}
}

func (s authPublicModuleService) HandleRoot(w http.ResponseWriter, r *http.Request) {
	s.handler.handleAppRoot(w, r)
}

func (s authPublicModuleService) HandleLogin(w http.ResponseWriter, r *http.Request) {
	s.handler.handleAppLoginPage(w, r, s.appName)
}

func (s authPublicModuleService) HandleAuthLogin(w http.ResponseWriter, r *http.Request) {
	s.handler.handleAuthLogin(w, r)
}

func (s authPublicModuleService) HandleAuthCallback(w http.ResponseWriter, r *http.Request) {
	s.handler.handleAuthCallback(w, r)
}

func (s authPublicModuleService) HandleAuthLogout(w http.ResponseWriter, r *http.Request) {
	s.handler.handleAuthLogout(w, r)
}

func (s authPublicModuleService) HandleMagicLink(w http.ResponseWriter, r *http.Request) {
	s.handler.handleMagicLink(w, r)
}

func (s authPublicModuleService) HandlePasskeyRegisterStart(w http.ResponseWriter, r *http.Request) {
	s.handler.handlePasskeyRegisterStart(w, r)
}

func (s authPublicModuleService) HandlePasskeyRegisterFinish(w http.ResponseWriter, r *http.Request) {
	s.handler.handlePasskeyRegisterFinish(w, r)
}

func (s authPublicModuleService) HandlePasskeyLoginStart(w http.ResponseWriter, r *http.Request) {
	s.handler.handlePasskeyLoginStart(w, r)
}

func (s authPublicModuleService) HandlePasskeyLoginFinish(w http.ResponseWriter, r *http.Request) {
	s.handler.handlePasskeyLoginFinish(w, r)
}

func (s authPublicModuleService) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func (h *handler) handleAppLoginPage(w http.ResponseWriter, r *http.Request, appName string) {
	if r.Method != http.MethodGet {
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}

	printer, lang := localizer(w, r)

	pendingID := strings.TrimSpace(r.URL.Query().Get("pending_id"))
	if pendingID == "" {
		if strings.TrimSpace(h.config.OAuthClientID) != "" {
			http.Redirect(w, r, routepath.AuthLogin, http.StatusFound)
			return
		}
		localizeHTTPError(w, r, http.StatusBadRequest, "error.http.pending_id_is_required")
		return
	}
	clientID := strings.TrimSpace(r.URL.Query().Get("client_id"))
	clientName := strings.TrimSpace(r.URL.Query().Get("client_name"))
	errorMessage := strings.TrimSpace(r.URL.Query().Get("error"))
	if clientName == "" {
		if clientID != "" {
			clientName = clientID
		} else {
			clientName = webtemplates.T(printer, "web.login.unknown_client")
		}
	}

	params := webtemplates.LoginParams{
		AppName:      appName,
		PendingID:    pendingID,
		ClientName:   clientName,
		Error:        errorMessage,
		Lang:         lang,
		Loc:          printer,
		CurrentPath:  r.URL.Path,
		CurrentQuery: r.URL.RawQuery,
	}
	loginPage := webtemplates.PageContext{
		Lang:    lang,
		Loc:     printer,
		AppName: appName,
	}
	if err := h.writePage(w, r, webtemplates.LoginPage(params), composeHTMXTitleForPage(loginPage, "title.login")); err != nil {
		log.Printf("web: failed to render login page: %v", err)
		localizeHTTPError(w, r, http.StatusInternalServerError, "error.http.web_handler_unavailable")
		return
	}
}
