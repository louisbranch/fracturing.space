package web

import (
	"log"
	"net/http"
	"strings"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	connectionsv1 "github.com/louisbranch/fracturing.space/api/gen/go/connections/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	sharedroute "github.com/louisbranch/fracturing.space/internal/services/shared/route"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (h *handler) handleAppSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	if sessionFromRequest(r, h.sessions) == nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}

	page := h.pageContext(w, r)
	if err := h.writePage(
		w,
		r,
		webtemplates.SettingsPage(page),
		composeHTMXTitleForPage(page, "layout.settings"),
	); err != nil {
		localizeHTTPError(w, r, http.StatusInternalServerError, "error.http.web_handler_unavailable")
	}
}

func (h *handler) handleAppSettingsRoutes(w http.ResponseWriter, r *http.Request) {
	if sharedroute.RedirectTrailingSlash(w, r) {
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/settings/")
	parts := splitSettingsPathParts(path)
	if len(parts) == 1 && parts[0] == "username" {
		h.handleAppUsernameSettings(w, r)
		return
	}
	if len(parts) == 1 && parts[0] == "ai-keys" {
		h.handleAppAIKeys(w, r)
		return
	}
	if len(parts) == 3 && parts[0] == "ai-keys" && parts[2] == "revoke" {
		h.handleAppAIKeyRevoke(w, r, parts[1])
		return
	}
	http.NotFound(w, r)
}

func splitSettingsPathParts(path string) []string {
	rawParts := strings.Split(strings.TrimSpace(path), "/")
	parts := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		parts = append(parts, part)
	}
	return parts
}

func (h *handler) handleAppUsernameSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleAppUsernameSettingsGet(w, r)
	case http.MethodPost:
		h.handleAppUsernameSettingsPost(w, r)
	default:
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodPost)
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
	}
}

func (h *handler) handleAppUsernameSettingsGet(w http.ResponseWriter, r *http.Request) {
	sess := sessionFromRequest(r, h.sessions)
	if sess == nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}
	page := h.pageContext(w, r)
	if h.connectionsClient == nil {
		writeUsernameSettingsPage(w, r, page, webtemplates.UsernameSettingsPageState{
			ErrorMessage: webtemplates.T(page.Loc, "web.settings.username.error_connections_unavailable"),
		}, http.StatusOK)
		return
	}
	userID, ok := h.resolveSettingsUserID(w, r, sess, "Username settings unavailable")
	if !ok {
		return
	}

	ctx := grpcauthctx.WithUserID(r.Context(), userID)
	resp, err := h.connectionsClient.GetUsername(ctx, &connectionsv1.GetUsernameRequest{UserId: userID})
	if err != nil {
		if status.Code(err) != codes.NotFound {
			h.renderErrorPage(w, r, grpcErrorHTTPStatus(err, http.StatusBadGateway), "Username settings unavailable", "failed to load username")
			return
		}
	}
	state := webtemplates.UsernameSettingsPageState{}
	if resp != nil && resp.GetUsernameRecord() != nil {
		state.Username = strings.TrimSpace(resp.GetUsernameRecord().GetUsername())
	}
	writeUsernameSettingsPage(w, r, page, state, http.StatusOK)
}

func (h *handler) handleAppUsernameSettingsPost(w http.ResponseWriter, r *http.Request) {
	sess := sessionFromRequest(r, h.sessions)
	if sess == nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}
	if h.connectionsClient == nil {
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Username settings unavailable", "connections service is not configured")
		return
	}
	if err := r.ParseForm(); err != nil {
		h.renderErrorPage(w, r, http.StatusBadRequest, "Username settings unavailable", "failed to parse username form")
		return
	}
	page := h.pageContext(w, r)
	username := strings.TrimSpace(r.FormValue("username"))
	if username == "" {
		writeUsernameSettingsPage(w, r, page, webtemplates.UsernameSettingsPageState{
			ErrorMessage: webtemplates.T(page.Loc, "web.settings.username.error_required"),
		}, http.StatusBadRequest)
		return
	}
	userID, ok := h.resolveSettingsUserID(w, r, sess, "Username settings unavailable")
	if !ok {
		return
	}
	ctx := grpcauthctx.WithUserID(r.Context(), userID)
	_, err := h.connectionsClient.SetUsername(ctx, &connectionsv1.SetUsernameRequest{
		UserId:   userID,
		Username: username,
	})
	if err != nil {
		statusCode := grpcErrorHTTPStatus(err, http.StatusBadGateway)
		writeUsernameSettingsPage(w, r, page, webtemplates.UsernameSettingsPageState{
			Username:     username,
			ErrorMessage: grpcErrorMessage(err, webtemplates.T(page.Loc, "web.settings.username.error_save_failed")),
		}, statusCode)
		return
	}
	http.Redirect(w, r, "/settings/username", http.StatusFound)
}

func (h *handler) handleAppAIKeys(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleAppAIKeysGet(w, r)
	case http.MethodPost:
		h.handleAppAIKeysCreate(w, r)
	default:
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodPost)
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
	}
}

func (h *handler) handleAppAIKeysGet(w http.ResponseWriter, r *http.Request) {
	sess := sessionFromRequest(r, h.sessions)
	if sess == nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}
	page := h.pageContext(w, r)
	if h.credentialClient == nil {
		renderAIKeysPage(w, r, page, webtemplates.AIKeysPageState{
			FormProvider: providerDisplayLabel(page.Loc, aiv1.Provider_PROVIDER_OPENAI),
			ErrorMessage: webtemplates.T(page.Loc, "web.settings.ai_keys.warning_unavailable"),
		}, http.StatusOK)
		return
	}

	userID, ok := h.resolveSettingsUserID(w, r, sess, "AI keys unavailable")
	if !ok {
		return
	}

	ctx := grpcauthctx.WithUserID(r.Context(), userID)
	resp, err := h.credentialClient.ListCredentials(ctx, &aiv1.ListCredentialsRequest{PageSize: 50})
	if err != nil {
		log.Printf("list ai credentials failed: user_id=%s err=%v", userID, err)
		renderAIKeysPage(w, r, page, webtemplates.AIKeysPageState{
			FormProvider: providerDisplayLabel(page.Loc, aiv1.Provider_PROVIDER_OPENAI),
			ErrorMessage: webtemplates.T(page.Loc, "web.settings.ai_keys.warning_unavailable"),
		}, http.StatusOK)
		return
	}

	renderAIKeysPage(w, r, page, webtemplates.AIKeysPageState{
		FormProvider: providerDisplayLabel(page.Loc, aiv1.Provider_PROVIDER_OPENAI),
		Keys:         toAIKeyRows(page.Loc, resp.GetCredentials()),
	}, http.StatusOK)
}

func (h *handler) handleAppAIKeysCreate(w http.ResponseWriter, r *http.Request) {
	sess := sessionFromRequest(r, h.sessions)
	if sess == nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}
	if h.credentialClient == nil {
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "AI key action unavailable", "credential service client is not configured")
		return
	}
	if err := r.ParseForm(); err != nil {
		h.renderErrorPage(w, r, http.StatusBadRequest, "AI key action unavailable", "failed to parse ai key form")
		return
	}

	page := h.pageContext(w, r)
	label := strings.TrimSpace(r.FormValue("label"))
	secret := strings.TrimSpace(r.FormValue("secret"))
	if label == "" || secret == "" {
		renderAIKeysPage(w, r, page, webtemplates.AIKeysPageState{
			FormLabel:    label,
			FormProvider: providerDisplayLabel(page.Loc, aiv1.Provider_PROVIDER_OPENAI),
			ErrorMessage: webtemplates.T(page.Loc, "web.settings.ai_keys.error_required"),
		}, http.StatusBadRequest)
		return
	}

	userID, ok := h.resolveSettingsUserID(w, r, sess, "AI key action unavailable")
	if !ok {
		return
	}

	ctx := grpcauthctx.WithUserID(r.Context(), userID)
	_, err := h.credentialClient.CreateCredential(ctx, &aiv1.CreateCredentialRequest{
		Provider: aiv1.Provider_PROVIDER_OPENAI,
		Label:    label,
		Secret:   secret,
	})
	if err != nil {
		renderAIKeysPage(w, r, page, webtemplates.AIKeysPageState{
			FormLabel:    label,
			FormProvider: providerDisplayLabel(page.Loc, aiv1.Provider_PROVIDER_OPENAI),
			ErrorMessage: grpcErrorMessage(err, webtemplates.T(page.Loc, "web.settings.ai_keys.error_create_failed")),
		}, grpcErrorHTTPStatus(err, http.StatusBadGateway))
		return
	}

	http.Redirect(w, r, "/settings/ai-keys", http.StatusFound)
}

func (h *handler) handleAppAIKeyRevoke(w http.ResponseWriter, r *http.Request, credentialID string) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	sess := sessionFromRequest(r, h.sessions)
	if sess == nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}
	if h.credentialClient == nil {
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "AI key action unavailable", "credential service client is not configured")
		return
	}

	credentialID = strings.TrimSpace(credentialID)
	if credentialID == "" {
		h.renderErrorPage(w, r, http.StatusBadRequest, "AI key action unavailable", "ai key id is required")
		return
	}

	userID, ok := h.resolveSettingsUserID(w, r, sess, "AI key action unavailable")
	if !ok {
		return
	}

	ctx := grpcauthctx.WithUserID(r.Context(), userID)
	_, err := h.credentialClient.RevokeCredential(ctx, &aiv1.RevokeCredentialRequest{
		CredentialId: credentialID,
	})
	if err != nil {
		h.renderErrorPage(w, r, grpcErrorHTTPStatus(err, http.StatusBadGateway), "AI key action unavailable", "failed to revoke ai key")
		return
	}
	http.Redirect(w, r, "/settings/ai-keys", http.StatusFound)
}

func (h *handler) resolveSettingsUserID(w http.ResponseWriter, r *http.Request, sess *session, title string) (string, bool) {
	userID, err := h.resolveProfileUserID(r.Context(), sess)
	if err != nil {
		h.renderErrorPage(w, r, http.StatusBadGateway, title, "failed to resolve current user")
		return "", false
	}
	if strings.TrimSpace(userID) == "" {
		h.renderErrorPage(w, r, http.StatusUnauthorized, "Authentication required", "no user identity was resolved for this session")
		return "", false
	}
	return userID, true
}

func renderAIKeysPage(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, state webtemplates.AIKeysPageState, statusCode int) {
	writeGameContentType(w)
	w.WriteHeader(statusCode)
	if err := writePage(w, r, webtemplates.AIKeysPage(page, state), composeHTMXTitleForPage(page, "layout.settings_ai_keys")); err != nil {
		localizeHTTPError(w, r, http.StatusInternalServerError, "error.http.web_handler_unavailable")
	}
}

func writeUsernameSettingsPage(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, state webtemplates.UsernameSettingsPageState, statusCode int) {
	writeGameContentType(w)
	w.WriteHeader(statusCode)
	if err := writePage(w, r, webtemplates.UsernameSettingsPage(page, state), composeHTMXTitleForPage(page, "layout.settings_username")); err != nil {
		localizeHTTPError(w, r, http.StatusInternalServerError, "error.http.web_handler_unavailable")
	}
}

func toAIKeyRows(loc webtemplates.Localizer, credentials []*aiv1.Credential) []webtemplates.AIKeyRow {
	rows := make([]webtemplates.AIKeyRow, 0, len(credentials))
	for _, credential := range credentials {
		if credential == nil {
			continue
		}
		credentialID := strings.TrimSpace(credential.GetId())
		statusValue := credential.GetStatus()
		safeCredentialID := credentialID
		canRevoke := credentialID != "" && statusValue == aiv1.CredentialStatus_CREDENTIAL_STATUS_ACTIVE
		if !isSafeCredentialPathID(credentialID) {
			safeCredentialID = ""
			canRevoke = false
		}
		rows = append(rows, webtemplates.AIKeyRow{
			ID:        safeCredentialID,
			Label:     strings.TrimSpace(credential.GetLabel()),
			Provider:  providerDisplayLabel(loc, credential.GetProvider()),
			Status:    credentialStatusDisplayLabel(loc, statusValue),
			CreatedAt: formatProtoTimestamp(credential.GetCreatedAt()),
			RevokedAt: formatProtoTimestamp(credential.GetRevokedAt()),
			CanRevoke: canRevoke,
		})
	}
	return rows
}

func isSafeCredentialPathID(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	return !strings.Contains(value, "/") && !strings.Contains(value, "\\")
}

func providerDisplayLabel(loc webtemplates.Localizer, provider aiv1.Provider) string {
	switch provider {
	case aiv1.Provider_PROVIDER_OPENAI:
		return "OpenAI"
	default:
		return webtemplates.T(loc, "web.settings.ai_keys.provider_unknown")
	}
}

func credentialStatusDisplayLabel(loc webtemplates.Localizer, statusValue aiv1.CredentialStatus) string {
	switch statusValue {
	case aiv1.CredentialStatus_CREDENTIAL_STATUS_ACTIVE:
		return webtemplates.T(loc, "web.settings.ai_keys.status_active")
	case aiv1.CredentialStatus_CREDENTIAL_STATUS_REVOKED:
		return webtemplates.T(loc, "web.settings.ai_keys.status_revoked")
	default:
		return webtemplates.T(loc, "web.settings.ai_keys.status_unspecified")
	}
}

func formatProtoTimestamp(value *timestamppb.Timestamp) string {
	if value == nil {
		return "-"
	}
	if err := value.CheckValid(); err != nil {
		return "-"
	}
	return value.AsTime().UTC().Format("2006-01-02 15:04 UTC")
}

func grpcErrorMessage(err error, fallback string) string {
	fallback = strings.TrimSpace(fallback)
	if fallback == "" {
		fallback = "request failed"
	}
	statusErr, ok := status.FromError(err)
	if !ok {
		return fallback
	}
	msg := strings.TrimSpace(statusErr.Message())
	if msg == "" {
		return fallback
	}
	return msg
}
