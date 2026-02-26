package publicprofile

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	connectionsv1 "github.com/louisbranch/fracturing.space/api/gen/go/connections/v1"
	routepath "github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	"github.com/louisbranch/fracturing.space/internal/services/web/support"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PublicProfileHandlers maps dependencies required for public profile rendering.
type PublicProfileHandlers struct {
	LookupProfile func(context.Context, *connectionsv1.LookupUserProfileRequest) (*connectionsv1.LookupUserProfileResponse, error)

	PageContext func(*http.Request) webtemplates.PageContext

	RenderErrorPage func(w http.ResponseWriter, r *http.Request, status int, title string, message string)
}

type publicProfilePageState struct {
	Username    string
	DisplayName string
	Bio         string
}

// HandlePublicProfile renders /u/:username when all dependencies are available.
func HandlePublicProfile(deps PublicProfileHandlers, w http.ResponseWriter, r *http.Request) {
	lookup := deps.LookupProfile
	pageContext := deps.PageContext
	renderError := deps.RenderErrorPage

	if lookup == nil || pageContext == nil || renderError == nil {
		http.NotFound(w, r)
		return
	}

	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		support.LocalizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}

	username := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, routepath.UserProfilePrefix))
	if username == "" || strings.Contains(username, "/") {
		http.NotFound(w, r)
		return
	}

	// The package path behavior is preserved from the web root implementation.
	resp, err := lookup(r.Context(), &connectionsv1.LookupUserProfileRequest{
		Username: username,
	})
	if err != nil {
		if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.NotFound {
			http.NotFound(w, r)
			return
		}
		renderError(w, r, http.StatusBadGateway, "Public profile unavailable", "failed to load public profile")
		return
	}

	profileRecord := resp.GetUserProfile()
	if profileRecord == nil {
		http.NotFound(w, r)
		return
	}

	resolvedUsername := strings.TrimSpace(profileRecord.GetUsername())
	if resolvedUsername == "" {
		resolvedUsername = username
	}
	displayName := strings.TrimSpace(profileRecord.GetName())
	if displayName == "" {
		displayName = "@" + resolvedUsername
	}

	page := pageContext(r)
	if err := renderPublicProfilePage(w, r, page, publicProfilePageState{
		Username:    resolvedUsername,
		DisplayName: displayName,
		Bio:         strings.TrimSpace(profileRecord.GetBio()),
	}); err != nil {
		support.LocalizeHTTPError(w, r, http.StatusInternalServerError, "error.http.web_handler_unavailable")
	}
}

func renderPublicProfilePage(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, state publicProfilePageState) error {
	return renderPublicShellPage(w, r, page, state.DisplayName, templ.ComponentFunc(func(_ context.Context, out io.Writer) error {
		escape := template.HTMLEscapeString
		if _, err := fmt.Fprintf(out, `<main class="landing-shell"><section class="landing-hero"><p class="hero-tagline">@%s</p><h1>%s</h1>`, escape(state.Username), escape(state.DisplayName)); err != nil {
			return err
		}
		if state.Bio != "" {
			if _, err := fmt.Fprintf(out, `<p class="hero-user">%s</p>`, escape(state.Bio)); err != nil {
				return err
			}
		}
		_, err := io.WriteString(out, `</section></main>`)
		return err
	}))
}

func renderPublicShellPage(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, title string, body templ.Component) error {
	if body == nil {
		return support.ErrNoWebPageComponent
	}
	shell := templ.ComponentFunc(func(ctx context.Context, out io.Writer) error {
		return webtemplates.ShellLayout(title, page).Render(templ.WithChildren(ctx, body), out)
	})
	return support.WritePage(w, r, shell, support.ComposeHTMXTitle(page.Loc, title))
}
