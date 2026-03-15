package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules"
	"github.com/louisbranch/fracturing.space/internal/services/web/principal"
)

func TestPrivateSettingsUsesAuthenticatedUserLocaleForShellAndContent(t *testing.T) {
	t.Parallel()

	account := &fakeAccountClient{getProfileResp: &authv1.GetProfileResponse{Profile: &authv1.AccountProfile{Locale: commonv1.Locale_LOCALE_PT_BR}}}
	auth := newFakeWebAuthClient()
	h, err := NewHandler(Config{
		Dependencies: newDependencyBundle(
			principal.Dependencies{SessionClient: auth, AccountClient: account},
			modules.Dependencies{
				PublicAuth: modules.PublicAuthDependencies{AuthClient: auth},
				Campaigns:  modules.CampaignDependencies{CampaignClient: defaultCampaignClient(), InteractionClient: defaultInteractionClient()},
				Profile:    modules.ProfileDependencies{SocialClient: defaultSocialClient()},
				Settings: modules.SettingsDependencies{
					SocialClient:     defaultSocialClient(),
					AccountClient:    account,
					CredentialClient: fakeCredentialClient{},
					AgentClient:      fakeAgentClient{},
				},
			},
		),
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/app/settings/profile", nil)
	attachSessionCookie(t, req, auth, "user-1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if !account.getProfileCalled {
		t.Fatalf("expected account profile lookup for authenticated locale")
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`<html lang="pt-BR"`,
		`>Configurações</a>`,
		`>Campanhas</a>`,
		`<h1 class="mb-0">Configurações</h1>`,
		`<h2 class="card-title">Perfil</h2>`,
		`<span class="label-text">Nome de usuário</span>`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing pt-BR marker %q: %q", marker, body)
		}
	}
}

func TestPrivateSettingsValidationErrorUsesAuthenticatedUserLocale(t *testing.T) {
	t.Parallel()

	account := &fakeAccountClient{getProfileResp: &authv1.GetProfileResponse{Profile: &authv1.AccountProfile{Locale: commonv1.Locale_LOCALE_PT_BR}}}
	auth := newFakeWebAuthClient()
	h, err := NewHandler(Config{
		Dependencies: newDependencyBundle(
			principal.Dependencies{SessionClient: auth, AccountClient: account},
			modules.Dependencies{
				PublicAuth: modules.PublicAuthDependencies{AuthClient: auth},
				Campaigns:  modules.CampaignDependencies{CampaignClient: defaultCampaignClient(), InteractionClient: defaultInteractionClient()},
				Profile:    modules.ProfileDependencies{SocialClient: defaultSocialClient()},
				Settings: modules.SettingsDependencies{
					SocialClient:     defaultSocialClient(),
					AccountClient:    account,
					CredentialClient: fakeCredentialClient{},
					AgentClient:      fakeAgentClient{},
				},
			},
		),
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	form := url.Values{"name": {"Rhea"}}
	form.Set("name", strings.Repeat("x", 65))
	req := httptest.NewRequest(http.MethodPost, "/app/settings/profile", strings.NewReader(form.Encode()))
	attachSessionCookie(t, req, auth, "user-1")
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`<html lang="pt-BR"`,
		"Nome deve ter no máximo 64 caracteres.",
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing pt-BR validation marker %q: %q", marker, body)
		}
	}
}

func TestLoginPageLocaleMenuUsesConsistentLabels(t *testing.T) {
	t.Parallel()

	h, err := NewHandler(Config{
		Dependencies: newDependencyBundle(
			principal.Dependencies{},
			modules.Dependencies{PublicAuth: modules.PublicAuthDependencies{AuthClient: newFakeWebAuthClient()}},
		),
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	tests := []struct {
		path        string
		wantTrigger string
	}{
		{path: "/login?lang=en-US", wantTrigger: "EN"},
		{path: "/login?lang=pt-BR", wantTrigger: "PT-BR"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
			}
			body := rr.Body.String()
			if !strings.Contains(body, `id="auth-language-menu"`) {
				t.Fatalf("body missing auth language menu: %q", body)
			}
			if !strings.Contains(body, `backdrop-blur">`+tc.wantTrigger+`</div>`) {
				t.Fatalf("body missing locale trigger label %q: %q", tc.wantTrigger, body)
			}
			for _, marker := range []string{`data-lang="en-US"`, `>EN</a>`, `data-lang="pt-BR"`, `>PT-BR</a>`} {
				if !strings.Contains(body, marker) {
					t.Fatalf("body missing locale option marker %q: %q", marker, body)
				}
			}
		})
	}
}
