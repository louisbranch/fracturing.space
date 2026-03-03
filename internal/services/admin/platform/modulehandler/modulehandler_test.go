package modulehandler

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/a-h/templ"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/requestctx"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"google.golang.org/grpc/metadata"
)

type fakeClientProvider struct {
	authCalled         bool
	accountCalled      bool
	campaignCalled     bool
	sessionCalled      bool
	characterCalled    bool
	participantCalled  bool
	inviteCalled       bool
	snapshotCalled     bool
	eventCalled        bool
	statsCalled        bool
	systemCalled       bool
	contentClientCalle bool
}

func (f *fakeClientProvider) AuthClient() authv1.AuthServiceClient {
	f.authCalled = true
	return nil
}

func (f *fakeClientProvider) AccountClient() authv1.AccountServiceClient {
	f.accountCalled = true
	return nil
}

func (f *fakeClientProvider) CampaignClient() statev1.CampaignServiceClient {
	f.campaignCalled = true
	return nil
}

func (f *fakeClientProvider) SessionClient() statev1.SessionServiceClient {
	f.sessionCalled = true
	return nil
}

func (f *fakeClientProvider) CharacterClient() statev1.CharacterServiceClient {
	f.characterCalled = true
	return nil
}

func (f *fakeClientProvider) ParticipantClient() statev1.ParticipantServiceClient {
	f.participantCalled = true
	return nil
}

func (f *fakeClientProvider) InviteClient() statev1.InviteServiceClient {
	f.inviteCalled = true
	return nil
}

func (f *fakeClientProvider) SnapshotClient() statev1.SnapshotServiceClient {
	f.snapshotCalled = true
	return nil
}

func (f *fakeClientProvider) EventClient() statev1.EventServiceClient {
	f.eventCalled = true
	return nil
}

func (f *fakeClientProvider) StatisticsClient() statev1.StatisticsServiceClient {
	f.statsCalled = true
	return nil
}

func (f *fakeClientProvider) SystemClient() statev1.SystemServiceClient {
	f.systemCalled = true
	return nil
}

func (f *fakeClientProvider) DaggerheartContentClient() daggerheartv1.DaggerheartContentServiceClient {
	f.contentClientCalle = true
	return nil
}

func TestBaseLocalizerAndContext(t *testing.T) {
	base := NewBase(nil)
	req := httptest.NewRequest(http.MethodGet, "/app/dashboard?lang=en-US", nil)
	rec := httptest.NewRecorder()

	loc, lang := base.Localizer(rec, req)
	if loc == nil {
		t.Fatal("Localizer() returned nil localizer")
	}
	if lang == "" {
		t.Fatal("Localizer() returned empty language")
	}
	if got := rec.Header().Get("Set-Cookie"); got == "" {
		t.Fatal("Localizer() did not persist language cookie")
	}

	page := base.PageContext(lang, loc, req)
	if page.CurrentPath != "/app/dashboard" || page.CurrentQuery != "lang=en-US" {
		t.Fatalf("PageContext() = %#v", page)
	}
}

func TestBaseGameContextAndHelpers(t *testing.T) {
	base := NewBase(nil)

	ctx, cancel := base.GameGRPCCallContext(requestctx.WithUserID(context.Background(), "user-1"))
	defer cancel()

	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		t.Fatal("GameGRPCCallContext() did not produce outgoing metadata")
	}
	values := md.Get(grpcmeta.UserIDHeader)
	if len(values) == 0 || values[0] != "user-1" {
		t.Fatalf("GameGRPCCallContext() metadata user id = %#v", values)
	}

	ctx, cancel = base.GameGRPCCallContext(nil)
	cancel()
	if ctx == nil {
		t.Fatal("GameGRPCCallContext(nil) returned nil context")
	}

	htmxReq := httptest.NewRequest(http.MethodGet, "/app/dashboard", nil)
	htmxReq.Header.Set("HX-Request", "true")
	if !base.IsHTMXRequest(htmxReq) {
		t.Fatal("IsHTMXRequest() did not detect HTMX header")
	}
	if base.IsHTMXRequest(httptest.NewRequest(http.MethodGet, "/app/dashboard", nil)) {
		t.Fatal("IsHTMXRequest() returned true without HTMX header")
	}

	title := base.HTMXLocalizedPageTitle(nil, "title.dashboard")
	if !strings.Contains(title, "<title>") {
		t.Fatalf("HTMXLocalizedPageTitle() = %q", title)
	}
}

func TestBaseRenderPageAndAccessors(t *testing.T) {
	provider := &fakeClientProvider{}
	base := NewBase(provider)

	component := templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		_, err := io.WriteString(w, "<main>content</main>")
		return err
	})

	req := httptest.NewRequest(http.MethodGet, "/app/dashboard", nil)
	rec := httptest.NewRecorder()
	base.RenderPage(rec, req, component, component, "<title>Demo</title>")
	if !strings.Contains(rec.Body.String(), "content") {
		t.Fatalf("RenderPage() body = %q", rec.Body.String())
	}

	_ = base.AuthClient()
	_ = base.AccountClient()
	_ = base.CampaignClient()
	_ = base.SessionClient()
	_ = base.CharacterClient()
	_ = base.ParticipantClient()
	_ = base.InviteClient()
	_ = base.SnapshotClient()
	_ = base.EventClient()
	_ = base.StatisticsClient()
	_ = base.SystemClient()
	_ = base.DaggerheartContentClient()

	if !provider.authCalled ||
		!provider.accountCalled ||
		!provider.campaignCalled ||
		!provider.sessionCalled ||
		!provider.characterCalled ||
		!provider.participantCalled ||
		!provider.inviteCalled ||
		!provider.snapshotCalled ||
		!provider.eventCalled ||
		!provider.statsCalled ||
		!provider.systemCalled ||
		!provider.contentClientCalle {
		t.Fatal("expected all provider accessors to be called")
	}

	nilBase := NewBase(nil)
	if nilBase.AuthClient() != nil ||
		nilBase.AccountClient() != nil ||
		nilBase.CampaignClient() != nil ||
		nilBase.SessionClient() != nil ||
		nilBase.CharacterClient() != nil ||
		nilBase.ParticipantClient() != nil ||
		nilBase.InviteClient() != nil ||
		nilBase.SnapshotClient() != nil ||
		nilBase.EventClient() != nil ||
		nilBase.StatisticsClient() != nil ||
		nilBase.SystemClient() != nil ||
		nilBase.DaggerheartContentClient() != nil {
		t.Fatal("nil provider accessors should return nil clients")
	}
}
