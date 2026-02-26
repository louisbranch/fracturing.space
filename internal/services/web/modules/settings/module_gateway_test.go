package settings

import (
	"context"
	"net/http"
	"testing"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestGRPCGatewayLoadAndSaveProfile(t *testing.T) {
	t.Parallel()

	social := &socialClientStub{
		getResp: &socialv1.GetUserProfileResponse{UserProfile: &socialv1.UserProfile{
			Username:      "  rhea  ",
			Name:          "  Rhea Vale  ",
			AvatarSetId:   "  set-a  ",
			AvatarAssetId: "  asset-1  ",
			Bio:           "  Traveler  ",
		}},
	}
	gateway := grpcGateway{socialClient: social}

	profile, err := gateway.LoadProfile(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("LoadProfile() error = %v", err)
	}
	if profile.Username != "rhea" {
		t.Fatalf("Username = %q, want %q", profile.Username, "rhea")
	}
	if profile.Name != "Rhea Vale" {
		t.Fatalf("Name = %q, want %q", profile.Name, "Rhea Vale")
	}

	err = gateway.SaveProfile(context.Background(), "user-1", SettingsProfile{
		Username:      "rhea",
		Name:          "Rhea Vale",
		AvatarSetID:   "set-a",
		AvatarAssetID: "asset-1",
		Bio:           "Traveler",
	})
	if err != nil {
		t.Fatalf("SaveProfile() error = %v", err)
	}
	if social.lastSetReq.GetUserId() != "user-1" {
		t.Fatalf("SetUserProfile user id = %q, want %q", social.lastSetReq.GetUserId(), "user-1")
	}
	if social.lastSetReq.GetUsername() != "rhea" {
		t.Fatalf("SetUserProfile username = %q, want %q", social.lastSetReq.GetUsername(), "rhea")
	}
}

func TestGRPCGatewayLoadProfileReturnsEmptyWhenSocialProfileMissing(t *testing.T) {
	t.Parallel()

	social := &socialClientStub{getErr: status.Error(codes.NotFound, "user profile not found")}
	gateway := grpcGateway{socialClient: social}

	profile, err := gateway.LoadProfile(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("LoadProfile() error = %v", err)
	}
	if profile != (SettingsProfile{}) {
		t.Fatalf("profile = %#v, want empty profile", profile)
	}
}

func TestGRPCGatewayLoadAndSaveLocale(t *testing.T) {
	t.Parallel()

	account := &accountClientStub{getResp: &authv1.GetProfileResponse{Profile: &authv1.AccountProfile{Locale: commonv1.Locale_LOCALE_PT_BR}}}
	gateway := grpcGateway{accountClient: account}

	locale, err := gateway.LoadLocale(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("LoadLocale() error = %v", err)
	}
	if locale != commonv1.Locale_LOCALE_PT_BR {
		t.Fatalf("locale = %v, want %v", locale, commonv1.Locale_LOCALE_PT_BR)
	}

	err = gateway.SaveLocale(context.Background(), "user-1", commonv1.Locale_LOCALE_EN_US)
	if err != nil {
		t.Fatalf("SaveLocale() error = %v", err)
	}
	if account.lastUpdateReq.GetUserId() != "user-1" {
		t.Fatalf("UpdateProfile user id = %q, want %q", account.lastUpdateReq.GetUserId(), "user-1")
	}
	if account.lastUpdateReq.GetLocale() != commonv1.Locale_LOCALE_EN_US {
		t.Fatalf("UpdateProfile locale = %v, want %v", account.lastUpdateReq.GetLocale(), commonv1.Locale_LOCALE_EN_US)
	}
}

func TestGRPCGatewayListCreateAndRevokeAIKeys(t *testing.T) {
	t.Parallel()

	created := timestamppb.New(time.Date(2026, 1, 2, 3, 4, 0, 0, time.UTC))
	credentials := &credentialClientStub{listResp: &aiv1.ListCredentialsResponse{Credentials: []*aiv1.Credential{
		{
			Id:        "cred-1",
			Label:     "Primary",
			Provider:  aiv1.Provider_PROVIDER_OPENAI,
			Status:    aiv1.CredentialStatus_CREDENTIAL_STATUS_ACTIVE,
			CreatedAt: created,
		},
		{
			Id:     "unsafe/id",
			Label:  "Unsafe",
			Status: aiv1.CredentialStatus_CREDENTIAL_STATUS_ACTIVE,
		},
		{
			Id:        "cred-3",
			Label:     "Unknown Provider",
			Provider:  aiv1.Provider_PROVIDER_UNSPECIFIED,
			Status:    aiv1.CredentialStatus_CREDENTIAL_STATUS_UNSPECIFIED,
			CreatedAt: &timestamppb.Timestamp{Seconds: 1, Nanos: 2_000_000_000},
		},
	}}}
	gateway := grpcGateway{credentialClient: credentials}

	rows, err := gateway.ListAIKeys(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListAIKeys() error = %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("len(rows) = %d, want 3", len(rows))
	}
	if rows[0].Provider != "OpenAI" {
		t.Fatalf("provider = %q, want %q", rows[0].Provider, "OpenAI")
	}
	if rows[0].Status != "Active" {
		t.Fatalf("status = %q, want %q", rows[0].Status, "Active")
	}
	if rows[0].CreatedAt != "2026-01-02 03:04 UTC" {
		t.Fatalf("created at = %q, want %q", rows[0].CreatedAt, "2026-01-02 03:04 UTC")
	}
	if !rows[0].CanRevoke {
		t.Fatalf("expected first key to be revocable")
	}
	if rows[1].ID != "" || rows[1].CanRevoke {
		t.Fatalf("expected unsafe key path id to be disabled: %#v", rows[1])
	}
	if rows[2].Provider != "Unknown" {
		t.Fatalf("provider = %q, want %q", rows[2].Provider, "Unknown")
	}
	if rows[2].Status != "Unspecified" {
		t.Fatalf("status = %q, want %q", rows[2].Status, "Unspecified")
	}
	if rows[2].CreatedAt != "-" {
		t.Fatalf("created at = %q, want %q", rows[2].CreatedAt, "-")
	}

	err = gateway.CreateAIKey(context.Background(), "user-1", "Primary", "sk-secret")
	if err != nil {
		t.Fatalf("CreateAIKey() error = %v", err)
	}
	if credentials.lastCreateReq.GetProvider() != aiv1.Provider_PROVIDER_OPENAI {
		t.Fatalf("provider = %v, want %v", credentials.lastCreateReq.GetProvider(), aiv1.Provider_PROVIDER_OPENAI)
	}
	if credentials.lastCreateReq.GetLabel() != "Primary" {
		t.Fatalf("label = %q, want %q", credentials.lastCreateReq.GetLabel(), "Primary")
	}

	err = gateway.RevokeAIKey(context.Background(), "user-1", "cred-1")
	if err != nil {
		t.Fatalf("RevokeAIKey() error = %v", err)
	}
	if credentials.lastRevokeReq.GetCredentialId() != "cred-1" {
		t.Fatalf("credential id = %q, want %q", credentials.lastRevokeReq.GetCredentialId(), "cred-1")
	}
}

func TestGRPCGatewayRequiresExplicitUserID(t *testing.T) {
	t.Parallel()

	gateway := grpcGateway{socialClient: &socialClientStub{}}

	_, err := gateway.LoadProfile(context.Background(), "   ")
	if err == nil {
		t.Fatalf("expected user-id error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusUnauthorized {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusUnauthorized)
	}
}

func TestGRPCGatewayMissingClientBehavior(t *testing.T) {
	t.Parallel()

	gateway := grpcGateway{}

	tests := []struct {
		name string
		run  func() error
	}{
		{name: "load profile", run: func() error { _, err := gateway.LoadProfile(context.Background(), "user-1"); return err }},
		{name: "save profile", run: func() error { return gateway.SaveProfile(context.Background(), "user-1", SettingsProfile{}) }},
		{name: "load locale", run: func() error { _, err := gateway.LoadLocale(context.Background(), "user-1"); return err }},
		{name: "save locale", run: func() error { return gateway.SaveLocale(context.Background(), "user-1", commonv1.Locale_LOCALE_EN_US) }},
		{name: "list ai keys", run: func() error { _, err := gateway.ListAIKeys(context.Background(), "user-1"); return err }},
		{name: "create ai key", run: func() error { return gateway.CreateAIKey(context.Background(), "user-1", "label", "secret") }},
		{name: "revoke ai key", run: func() error { return gateway.RevokeAIKey(context.Background(), "user-1", "cred-1") }},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.run()
			if err == nil {
				t.Fatalf("expected unavailable error")
			}
			if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
				t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
			}
		})
	}
}

type socialClientStub struct {
	getResp    *socialv1.GetUserProfileResponse
	getErr     error
	lookupResp *socialv1.LookupUserProfileResponse
	lookupErr  error
	setErr     error
	lastSetReq *socialv1.SetUserProfileRequest
}

func (f *socialClientStub) GetUserProfile(context.Context, *socialv1.GetUserProfileRequest, ...grpc.CallOption) (*socialv1.GetUserProfileResponse, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.getResp != nil {
		return f.getResp, nil
	}
	return &socialv1.GetUserProfileResponse{}, nil
}

func (f *socialClientStub) SetUserProfile(_ context.Context, req *socialv1.SetUserProfileRequest, _ ...grpc.CallOption) (*socialv1.SetUserProfileResponse, error) {
	f.lastSetReq = req
	if f.setErr != nil {
		return nil, f.setErr
	}
	return &socialv1.SetUserProfileResponse{}, nil
}

func (f *socialClientStub) LookupUserProfile(context.Context, *socialv1.LookupUserProfileRequest, ...grpc.CallOption) (*socialv1.LookupUserProfileResponse, error) {
	if f.lookupErr != nil {
		return nil, f.lookupErr
	}
	if f.lookupResp != nil {
		return f.lookupResp, nil
	}
	return &socialv1.LookupUserProfileResponse{}, nil
}

type accountClientStub struct {
	getResp       *authv1.GetProfileResponse
	getErr        error
	updateErr     error
	lastUpdateReq *authv1.UpdateProfileRequest
}

func (f *accountClientStub) GetProfile(context.Context, *authv1.GetProfileRequest, ...grpc.CallOption) (*authv1.GetProfileResponse, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.getResp != nil {
		return f.getResp, nil
	}
	return &authv1.GetProfileResponse{}, nil
}

func (f *accountClientStub) UpdateProfile(_ context.Context, req *authv1.UpdateProfileRequest, _ ...grpc.CallOption) (*authv1.UpdateProfileResponse, error) {
	f.lastUpdateReq = req
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	return &authv1.UpdateProfileResponse{}, nil
}

type credentialClientStub struct {
	listResp      *aiv1.ListCredentialsResponse
	listErr       error
	createErr     error
	revokeErr     error
	lastCreateReq *aiv1.CreateCredentialRequest
	lastRevokeReq *aiv1.RevokeCredentialRequest
}

func (f *credentialClientStub) ListCredentials(context.Context, *aiv1.ListCredentialsRequest, ...grpc.CallOption) (*aiv1.ListCredentialsResponse, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	if f.listResp != nil {
		return f.listResp, nil
	}
	return &aiv1.ListCredentialsResponse{}, nil
}

func (f *credentialClientStub) CreateCredential(_ context.Context, req *aiv1.CreateCredentialRequest, _ ...grpc.CallOption) (*aiv1.CreateCredentialResponse, error) {
	f.lastCreateReq = req
	if f.createErr != nil {
		return nil, f.createErr
	}
	return &aiv1.CreateCredentialResponse{}, nil
}

func (f *credentialClientStub) RevokeCredential(_ context.Context, req *aiv1.RevokeCredentialRequest, _ ...grpc.CallOption) (*aiv1.RevokeCredentialResponse, error) {
	f.lastRevokeReq = req
	if f.revokeErr != nil {
		return nil, f.revokeErr
	}
	return &aiv1.RevokeCredentialResponse{}, nil
}
