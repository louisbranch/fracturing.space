package settings

import (
	"context"
	"log"
	"net/http"
	"strings"
	"unicode/utf8"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	connectionsv1 "github.com/louisbranch/fracturing.space/api/gen/go/connections/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	"github.com/louisbranch/fracturing.space/internal/services/web/support"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// Keep these aligned with internal/services/connections/profile/user_profile.go.
	userProfileNameMaxLength = 64
)

type AppSettingsHandlers struct {
	Authenticate         func(*http.Request) bool
	RedirectToLogin      func(http.ResponseWriter, *http.Request)
	HasConnectionsClient func() bool
	HasCredentialClient  func() bool
	ResolveProfileUserID func(context.Context) (string, error)
	GetUserProfile       func(context.Context, *connectionsv1.GetUserProfileRequest) (*connectionsv1.GetUserProfileResponse, error)
	SetUserProfile       func(context.Context, *connectionsv1.SetUserProfileRequest) (*connectionsv1.SetUserProfileResponse, error)
	ListCredentials      func(context.Context, *aiv1.ListCredentialsRequest) (*aiv1.ListCredentialsResponse, error)
	CreateCredential     func(context.Context, *aiv1.CreateCredentialRequest) (*aiv1.CreateCredentialResponse, error)
	RevokeCredential     func(context.Context, *aiv1.RevokeCredentialRequest) (*aiv1.RevokeCredentialResponse, error)
	RenderErrorPage      func(http.ResponseWriter, *http.Request, int, string, string)
	PageContext          func(*http.Request) webtemplates.PageContext
}

func HandleAppSettings(h AppSettingsHandlers, w http.ResponseWriter, r *http.Request) {
	if !validateAppSettingsBaseHandlers(h, w, r) {
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		support.LocalizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	if !h.Authenticate(r) {
		h.RedirectToLogin(w, r)
		return
	}
	page := h.PageContext(r)
	if err := support.WritePage(
		w,
		r,
		webtemplates.SettingsPage(page),
		support.ComposeHTMXTitleForPage(page, "layout.settings"),
	); err != nil {
		support.LocalizeHTTPError(w, r, http.StatusInternalServerError, "error.http.web_handler_unavailable")
	}
}

func HandleAppUserProfileSettings(h AppSettingsHandlers, w http.ResponseWriter, r *http.Request) {
	if !validateAppSettingsBaseHandlers(h, w, r) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		handleAppUserProfileSettingsGet(h, w, r)
	case http.MethodPost:
		handleAppUserProfileSettingsPost(h, w, r)
	default:
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodPost)
		support.LocalizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
	}
}

func handleAppUserProfileSettingsGet(h AppSettingsHandlers, w http.ResponseWriter, r *http.Request) {
	if !validateAppSettingsUserProfileGetHandlers(h, w, r) {
		return
	}
	if !h.Authenticate(r) {
		h.RedirectToLogin(w, r)
		return
	}
	page := h.PageContext(r)
	if !h.HasConnectionsClient() {
		renderUserProfileSettingsPage(w, r, page, webtemplates.UserProfileSettingsPageState{
			ErrorMessage: webtemplates.T(page.Loc, "web.settings.user_profile.error_connections_unavailable"),
		}, http.StatusOK)
		return
	}
	userID, ok := resolveSettingsUserID(h, w, r, page, "User profile settings unavailable")
	if !ok {
		return
	}
	userCtx := grpcauthctx.WithUserID(r.Context(), userID)
	resp, err := h.GetUserProfile(userCtx, &connectionsv1.GetUserProfileRequest{UserId: userID})
	if err != nil {
		if status.Code(err) != codes.NotFound {
			h.RenderErrorPage(w, r, support.GRPCErrorHTTPStatus(err, http.StatusBadGateway), "User profile settings unavailable", "failed to load user profile")
			return
		}
	}
	state := webtemplates.UserProfileSettingsPageState{}
	if resp != nil && resp.GetUserProfile() != nil {
		record := resp.GetUserProfile()
		state.Username = strings.TrimSpace(record.GetUsername())
		state.Name = strings.TrimSpace(record.GetName())
		state.AvatarSetID = strings.TrimSpace(record.GetAvatarSetId())
		state.AvatarAssetID = strings.TrimSpace(record.GetAvatarAssetId())
		state.Bio = strings.TrimSpace(record.GetBio())
	}
	renderUserProfileSettingsPage(w, r, page, state, http.StatusOK)
}

func handleAppUserProfileSettingsPost(h AppSettingsHandlers, w http.ResponseWriter, r *http.Request) {
	if !validateAppSettingsUserProfilePostHandlers(h, w, r) {
		return
	}
	if !h.Authenticate(r) {
		h.RedirectToLogin(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		h.RenderErrorPage(w, r, http.StatusBadRequest, "User profile settings unavailable", "failed to parse user profile form")
		return
	}
	page := h.PageContext(r)
	state := webtemplates.UserProfileSettingsPageState{
		Username:      strings.TrimSpace(r.FormValue("username")),
		Name:          strings.TrimSpace(r.FormValue("name")),
		AvatarSetID:   strings.TrimSpace(r.FormValue("avatar_set_id")),
		AvatarAssetID: strings.TrimSpace(r.FormValue("avatar_asset_id")),
		Bio:           strings.TrimSpace(r.FormValue("bio")),
	}
	if !h.HasConnectionsClient() {
		state.ErrorMessage = webtemplates.T(page.Loc, "web.settings.user_profile.error_connections_unavailable")
		renderUserProfileSettingsPage(w, r, page, state, http.StatusOK)
		return
	}
	if state.Username == "" {
		state.ErrorMessage = webtemplates.T(page.Loc, "web.settings.user_profile.error_username_required")
		renderUserProfileSettingsPage(w, r, page, state, http.StatusBadRequest)
		return
	}
	if state.Name == "" {
		state.ErrorMessage = webtemplates.T(page.Loc, "web.settings.user_profile.error_name_required")
		renderUserProfileSettingsPage(w, r, page, state, http.StatusBadRequest)
		return
	}
	if utf8.RuneCountInString(state.Name) > userProfileNameMaxLength {
		state.ErrorMessage = webtemplates.T(page.Loc, "web.settings.user_profile.error_name_too_long")
		renderUserProfileSettingsPage(w, r, page, state, http.StatusBadRequest)
		return
	}
	userID, ok := resolveSettingsUserID(h, w, r, page, "User profile settings unavailable")
	if !ok {
		return
	}
	userCtx := grpcauthctx.WithUserID(r.Context(), userID)
	_, err := h.SetUserProfile(userCtx, &connectionsv1.SetUserProfileRequest{
		UserId:        userID,
		Username:      state.Username,
		Name:          state.Name,
		AvatarSetId:   state.AvatarSetID,
		AvatarAssetId: state.AvatarAssetID,
		Bio:           state.Bio,
	})
	if err != nil {
		state.ErrorMessage = grpcErrorMessage(err, webtemplates.T(page.Loc, "web.settings.user_profile.error_save_failed"))
		renderUserProfileSettingsPage(w, r, page, state, support.GRPCErrorHTTPStatus(err, http.StatusBadGateway))
		return
	}
	http.Redirect(w, r, routepath.AppSettingsPrefix+"user-profile", http.StatusFound)
}

func HandleAppAIKeys(h AppSettingsHandlers, w http.ResponseWriter, r *http.Request) {
	if !validateAppSettingsBaseHandlers(h, w, r) ||
		!validateAppSettingsAIKeyHandlers(h, w, r) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		handleAppAIKeysGet(h, w, r)
	case http.MethodPost:
		handleAppAIKeysCreate(h, w, r)
	default:
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodPost)
		support.LocalizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
	}
}

func handleAppAIKeysGet(h AppSettingsHandlers, w http.ResponseWriter, r *http.Request) {
	if !h.Authenticate(r) {
		h.RedirectToLogin(w, r)
		return
	}
	page := h.PageContext(r)
	if !h.HasCredentialClient() {
		renderAIKeysPage(w, r, page, webtemplates.AIKeysPageState{
			FormProvider: providerDisplayLabel(page.Loc, aiv1.Provider_PROVIDER_OPENAI),
			ErrorMessage: webtemplates.T(page.Loc, "web.settings.ai_keys.warning_unavailable"),
		}, http.StatusOK)
		return
	}

	userID, ok := resolveSettingsUserID(h, w, r, page, "AI keys unavailable")
	if !ok {
		return
	}
	userCtx := grpcauthctx.WithUserID(r.Context(), userID)
	resp, err := h.ListCredentials(userCtx, &aiv1.ListCredentialsRequest{PageSize: 50})
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
		Keys:         ToAIKeyRows(page.Loc, resp.GetCredentials()),
	}, http.StatusOK)
}

func handleAppAIKeysCreate(h AppSettingsHandlers, w http.ResponseWriter, r *http.Request) {
	if !h.Authenticate(r) {
		h.RedirectToLogin(w, r)
		return
	}
	if !h.HasCredentialClient() {
		h.RenderErrorPage(w, r, http.StatusServiceUnavailable, "AI key action unavailable", "credential service client is not configured")
		return
	}
	if err := r.ParseForm(); err != nil {
		h.RenderErrorPage(w, r, http.StatusBadRequest, "AI key action unavailable", "failed to parse ai key form")
		return
	}
	page := h.PageContext(r)
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
	userID, ok := resolveSettingsUserID(h, w, r, page, "AI key action unavailable")
	if !ok {
		return
	}
	userCtx := grpcauthctx.WithUserID(r.Context(), userID)
	if _, err := h.CreateCredential(userCtx, &aiv1.CreateCredentialRequest{
		Provider: aiv1.Provider_PROVIDER_OPENAI,
		Label:    label,
		Secret:   secret,
	}); err != nil {
		renderAIKeysPage(w, r, page, webtemplates.AIKeysPageState{
			FormLabel:    label,
			FormProvider: providerDisplayLabel(page.Loc, aiv1.Provider_PROVIDER_OPENAI),
			ErrorMessage: grpcErrorMessage(err, webtemplates.T(page.Loc, "web.settings.ai_keys.error_create_failed")),
		}, support.GRPCErrorHTTPStatus(err, http.StatusBadGateway))
		return
	}
	http.Redirect(w, r, routepath.AppSettingsPrefix+"ai-keys", http.StatusFound)
}

func HandleAppAIKeyRevoke(h AppSettingsHandlers, w http.ResponseWriter, r *http.Request, credentialID string) {
	if !validateAppSettingsBaseHandlers(h, w, r) ||
		!validateAppSettingsAIKeyRevokeHandlers(h, w, r) {
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		support.LocalizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	if !h.Authenticate(r) {
		h.RedirectToLogin(w, r)
		return
	}
	if !h.HasCredentialClient() {
		h.RenderErrorPage(w, r, http.StatusServiceUnavailable, "AI key action unavailable", "credential service client is not configured")
		return
	}
	credentialID = strings.TrimSpace(credentialID)
	if credentialID == "" {
		h.RenderErrorPage(w, r, http.StatusBadRequest, "AI key action unavailable", "ai key id is required")
		return
	}
	userID, ok := resolveSettingsUserID(h, w, r, h.PageContext(r), "AI key action unavailable")
	if !ok {
		return
	}
	userCtx := grpcauthctx.WithUserID(r.Context(), userID)
	_, err := h.RevokeCredential(userCtx, &aiv1.RevokeCredentialRequest{
		CredentialId: credentialID,
	})
	if err != nil {
		h.RenderErrorPage(w, r, support.GRPCErrorHTTPStatus(err, http.StatusBadGateway), "AI key action unavailable", "failed to revoke ai key")
		return
	}
	http.Redirect(w, r, routepath.AppSettingsPrefix+"ai-keys", http.StatusFound)
}

func resolveSettingsUserID(h AppSettingsHandlers, w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, title string) (string, bool) {
	userID, err := h.ResolveProfileUserID(r.Context())
	if err != nil {
		h.RenderErrorPage(w, r, http.StatusBadGateway, title, "failed to resolve current user")
		return "", false
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		h.RenderErrorPage(w, r, http.StatusUnauthorized, "Authentication required", "no user identity was resolved for this session")
		return "", false
	}
	return userID, true
}

func validateAppSettingsBaseHandlers(h AppSettingsHandlers, w http.ResponseWriter, r *http.Request) bool {
	if h.Authenticate == nil ||
		h.RedirectToLogin == nil ||
		h.ResolveProfileUserID == nil ||
		h.RenderErrorPage == nil ||
		h.PageContext == nil {
		http.NotFound(w, r)
		return false
	}
	return true
}

func validateAppSettingsUserProfileGetHandlers(h AppSettingsHandlers, w http.ResponseWriter, r *http.Request) bool {
	if h.HasConnectionsClient == nil ||
		h.GetUserProfile == nil {
		http.NotFound(w, r)
		return false
	}
	return true
}

func validateAppSettingsUserProfilePostHandlers(h AppSettingsHandlers, w http.ResponseWriter, r *http.Request) bool {
	if h.HasConnectionsClient == nil ||
		h.SetUserProfile == nil {
		http.NotFound(w, r)
		return false
	}
	return true
}

func validateAppSettingsAIKeyHandlers(h AppSettingsHandlers, w http.ResponseWriter, r *http.Request) bool {
	if h.HasCredentialClient == nil ||
		h.ListCredentials == nil ||
		h.CreateCredential == nil {
		http.NotFound(w, r)
		return false
	}
	return true
}

func validateAppSettingsAIKeyRevokeHandlers(h AppSettingsHandlers, w http.ResponseWriter, r *http.Request) bool {
	if h.HasCredentialClient == nil ||
		h.RevokeCredential == nil {
		http.NotFound(w, r)
		return false
	}
	return true
}

func renderAIKeysPage(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, state webtemplates.AIKeysPageState, statusCode int) {
	support.WriteGameContentType(w)
	w.WriteHeader(statusCode)
	if err := support.WritePage(
		w,
		r,
		webtemplates.AIKeysPage(page, state),
		support.ComposeHTMXTitleForPage(page, "layout.settings_ai_keys"),
	); err != nil {
		support.LocalizeHTTPError(w, r, http.StatusInternalServerError, "error.http.web_handler_unavailable")
	}
}

func renderUserProfileSettingsPage(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, state webtemplates.UserProfileSettingsPageState, statusCode int) {
	support.WriteGameContentType(w)
	w.WriteHeader(statusCode)
	if err := support.WritePage(
		w,
		r,
		webtemplates.UserProfileSettingsPage(page, state),
		support.ComposeHTMXTitleForPage(page, "layout.settings_user_profile"),
	); err != nil {
		support.LocalizeHTTPError(w, r, http.StatusInternalServerError, "error.http.web_handler_unavailable")
	}
}

func ToAIKeyRows(loc webtemplates.Localizer, credentials []*aiv1.Credential) []webtemplates.AIKeyRow {
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
