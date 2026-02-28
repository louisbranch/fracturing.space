package campaigns

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	"google.golang.org/grpc/metadata"
)

func TestMountCharacterCreateUsesHXRedirect(t *testing.T) {
	t.Parallel()
	m := NewStableWithGateway(managerMutationGateway(), managerMutationBase(), "", nil)
	mount, _ := m.Mount()
	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignCharacterCreate("c1"), strings.NewReader("name=Hero&kind=pc"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if got := rr.Header().Get("HX-Redirect"); got != routepath.AppCampaignCharacter("c1", "char-created") {
		t.Fatalf("HX-Redirect = %q, want %q", got, routepath.AppCampaignCharacter("c1", "char-created"))
	}
}

func TestMountCharacterCreateRedirectsForNonHTMX(t *testing.T) {
	t.Parallel()

	m := NewStableWithGateway(managerMutationGateway(), managerMutationBase(), "", nil)
	mount, _ := m.Mount()
	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignCharacterCreate("c1"), strings.NewReader("name=Hero&kind=pc"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != routepath.AppCampaignCharacter("c1", "char-created") {
		t.Fatalf("Location = %q, want %q", got, routepath.AppCampaignCharacter("c1", "char-created"))
	}
}

func TestMountCharacterCreateRejectsInvalidKind(t *testing.T) {
	t.Parallel()

	m := NewStableWithGateway(managerMutationGateway(), managerMutationBase(), "", nil)
	mount, _ := m.Mount()
	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignCharacterCreate("c1"), strings.NewReader("name=Hero&kind=invalid"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestRequestContextWithUserIDBehavior(t *testing.T) {
	t.Parallel()

	h := newHandlers(newService(fakeGateway{}), modulehandler.NewBase(nil, nil, nil), "")
	req := httptest.NewRequest(http.MethodGet, routepath.CampaignsPrefix, nil)
	ctx, _ := h.RequestContextAndUserID(req)
	if md, ok := metadata.FromOutgoingContext(ctx); ok && len(md.Get(grpcmeta.UserIDHeader)) > 0 {
		t.Fatalf("unexpected user metadata when resolver is nil")
	}

	h = newHandlers(newService(fakeGateway{}), modulehandler.NewBase(func(*http.Request) string { return "user-123" }, nil, nil), "")
	ctx, _ = h.RequestContextAndUserID(req)
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		t.Fatalf("expected outgoing metadata")
	}
	if got := md.Get(grpcmeta.UserIDHeader); len(got) != 1 || got[0] != "user-123" {
		t.Fatalf("user metadata = %v, want [user-123]", got)
	}
}

func TestParseAppCharacterKind(t *testing.T) {
	t.Parallel()

	if kind, ok := parseAppCharacterKind("pc"); !ok || kind != CharacterKindPC {
		t.Fatalf("parseAppCharacterKind pc = (%v, %v)", kind, ok)
	}
	if kind, ok := parseAppCharacterKind("npc"); !ok || kind != CharacterKindNPC {
		t.Fatalf("parseAppCharacterKind npc = (%v, %v)", kind, ok)
	}
	if _, ok := parseAppCharacterKind("invalid"); ok {
		t.Fatalf("expected invalid character kind to fail parse")
	}
}

func managerMutationGateway() fakeGateway {
	return fakeGateway{
		items: []CampaignSummary{{ID: "c1", Name: "First"}},
		participants: []CampaignParticipant{{
			ID:             "p-manager",
			UserID:         "user-123",
			CampaignAccess: "Manager",
		}},
	}
}

func managerMutationBase() modulehandler.Base {
	return modulehandler.NewBase(func(*http.Request) string { return "user-123" }, nil, nil)
}
