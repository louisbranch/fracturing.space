package modulehandler

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/a-h/templ"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/platform/requestctx"
	"google.golang.org/grpc/metadata"
)

func TestBaseLocalizerAndContext(t *testing.T) {
	base := NewBase()
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
	base := NewBase()

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

func TestBaseRenderPage(t *testing.T) {
	base := NewBase()

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
}
