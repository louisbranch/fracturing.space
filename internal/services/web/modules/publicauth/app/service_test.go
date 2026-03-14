package app

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth/redirectpath"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func TestPasskeyLoginStartRequiresUsername(t *testing.T) {
	svc := NewService(&authGatewayStub{}, "")
	_, err := svc.PasskeyLoginStart(context.Background(), "  ")
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestPasskeyRegisterStartAndFinish(t *testing.T) {
	stub := &authGatewayStub{
		beginRegistrationResp: PasskeyChallenge{SessionID: "reg-1", PublicKey: json.RawMessage(`{"publicKey":{}}`)},
		finishRegistrationResp: PasskeyFinish{
			SessionID:    "web-1",
			UserID:       "user-1",
			RecoveryCode: "ABCD-EFGH",
		},
	}
	svc := NewService(stub, "")

	start, err := svc.PasskeyRegisterStart(context.Background(), "louis")
	if err != nil {
		t.Fatalf("PasskeyRegisterStart() error = %v", err)
	}
	if start.SessionID != "reg-1" {
		t.Fatalf("SessionID = %q, want %q", start.SessionID, "reg-1")
	}

	finish, err := svc.PasskeyRegisterFinish(context.Background(), "reg-1", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("PasskeyRegisterFinish() error = %v", err)
	}
	if finish.SessionID != "web-1" || finish.UserID != "user-1" || finish.RecoveryCode != "ABCD-EFGH" {
		t.Fatalf("finish = %+v", finish)
	}
}

func TestPasskeyLoginFinishCreatesWebSession(t *testing.T) {
	stub := &authGatewayStub{
		finishLoginUserID: "user-1",
		webSessionID:      "web-1",
	}
	svc := NewService(stub, "")

	finish, err := svc.PasskeyLoginFinish(context.Background(), "login-1", json.RawMessage(`{}`), "")
	if err != nil {
		t.Fatalf("PasskeyLoginFinish() error = %v", err)
	}
	if finish.SessionID != "web-1" || finish.UserID != "user-1" {
		t.Fatalf("finish = %+v", finish)
	}
}

func TestCheckUsernameAvailabilityTreatsBlankAsInvalid(t *testing.T) {
	svc := NewService(&authGatewayStub{}, "")
	availability, err := svc.CheckUsernameAvailability(context.Background(), "   ")
	if err != nil {
		t.Fatalf("CheckUsernameAvailability() error = %v", err)
	}
	if availability.State != UsernameAvailabilityStateInvalid {
		t.Fatalf("state = %q, want %q", availability.State, UsernameAvailabilityStateInvalid)
	}
}

func TestCheckUsernameAvailabilityDelegatesToGateway(t *testing.T) {
	stub := &authGatewayStub{usernameAvailability: UsernameAvailability{CanonicalUsername: "louis", State: UsernameAvailabilityStateAvailable}}
	svc := NewService(stub, "")
	availability, err := svc.CheckUsernameAvailability(context.Background(), "Louis")
	if err != nil {
		t.Fatalf("CheckUsernameAvailability() error = %v", err)
	}
	if availability.CanonicalUsername != "louis" || availability.State != UsernameAvailabilityStateAvailable {
		t.Fatalf("availability = %+v", availability)
	}
}

func TestResolvePostAuthRedirectPathAllowsSafeInviteAndAppPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "dashboard path", raw: "/app/dashboard/", want: routepath.AppDashboard},
		{name: "invite path", raw: "/invite/inv-1", want: "/invite/inv-1"},
		{name: "invite path with query", raw: "/invite/inv-1?from=mail", want: "/invite/inv-1?from=mail"},
		{name: "slash normalized", raw: "invite/inv-1", want: "/invite/inv-1"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := redirectpath.ResolveSafe(tc.raw); got != tc.want {
				t.Fatalf("ResolveSafe(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

func TestResolvePostAuthRedirectPathRejectsUnsafeTargets(t *testing.T) {
	t.Parallel()

	tests := []string{
		"",
		"https://example.com/app/dashboard",
		"/invite",
		"/app",
		"/discover/campaigns",
		"/invite/%2fetc",
		"/invite/../admin",
	}

	for _, raw := range tests {
		if got := redirectpath.ResolveSafe(raw); got != "" {
			t.Fatalf("ResolveSafe(%q) = %q, want empty", raw, got)
		}
	}
}

type authGatewayStub struct {
	beginRegistrationResp  PasskeyChallenge
	usernameAvailability   UsernameAvailability
	finishRegistrationResp PasskeyFinish
	beginLoginResp         PasskeyChallenge
	finishLoginUserID      string
	webSessionID           string
}

func (f *authGatewayStub) BeginAccountRegistration(context.Context, string) (PasskeyChallenge, error) {
	return f.beginRegistrationResp, nil
}

func (f *authGatewayStub) CheckUsernameAvailability(context.Context, string) (UsernameAvailability, error) {
	return f.usernameAvailability, nil
}

func (f *authGatewayStub) FinishAccountRegistration(context.Context, string, json.RawMessage) (PasskeyFinish, error) {
	return f.finishRegistrationResp, nil
}

func (f *authGatewayStub) BeginPasskeyLogin(context.Context, string) (PasskeyChallenge, error) {
	return f.beginLoginResp, nil
}

func (f *authGatewayStub) FinishPasskeyLogin(context.Context, string, json.RawMessage, string) (string, error) {
	return f.finishLoginUserID, nil
}

func (*authGatewayStub) BeginAccountRecovery(context.Context, string, string) (string, error) {
	return "", nil
}

func (*authGatewayStub) BeginRecoveryPasskeyRegistration(context.Context, string) (PasskeyChallenge, error) {
	return PasskeyChallenge{}, nil
}

func (*authGatewayStub) FinishRecoveryPasskeyRegistration(context.Context, string, string, json.RawMessage, string) (PasskeyFinish, error) {
	return PasskeyFinish{}, nil
}

func (f *authGatewayStub) CreateWebSession(context.Context, string) (string, error) {
	return f.webSessionID, nil
}

func (*authGatewayStub) RevokeWebSession(context.Context, string) error {
	return nil
}
