package discovery

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	listingv1 "github.com/louisbranch/fracturing.space/api/gen/go/listing/v1"
	routepath "github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	"github.com/louisbranch/fracturing.space/internal/services/web/support"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DiscoveryHandlers maps dependencies required for discovery routes.
type DiscoveryHandlers struct {
	ListCampaignListings func(context.Context, *listingv1.ListCampaignListingsRequest, ...grpc.CallOption) (*listingv1.ListCampaignListingsResponse, error)
	GetCampaignListing   func(context.Context, *listingv1.GetCampaignListingRequest, ...grpc.CallOption) (*listingv1.GetCampaignListingResponse, error)

	PageContext     func(*http.Request) webtemplates.PageContext
	RenderErrorPage func(w http.ResponseWriter, r *http.Request, status int, title string, message string)
}

type publicListingCard struct {
	CampaignID       string
	Title            string
	Description      string
	ExpectedDuration string
}

// HandleDiscover renders /discover list pages.
func HandleDiscover(deps DiscoveryHandlers, w http.ResponseWriter, r *http.Request) {
	listCampaigns := deps.ListCampaignListings
	pageContext := deps.PageContext
	renderError := deps.RenderErrorPage

	if pageContext == nil || renderError == nil {
		http.NotFound(w, r)
		return
	}
	if r.URL.Path != "/discover" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		support.LocalizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}

	listings := []publicListingCard{}
	if listCampaigns != nil {
		resp, err := listCampaigns(r.Context(), &listingv1.ListCampaignListingsRequest{PageSize: 24})
		if err != nil {
			renderError(w, r, http.StatusBadGateway, "Discovery unavailable", "failed to list campaign discovery cards")
			return
		}
		listings = listingCardsFromProto(resp.GetListings())
	}

	page := pageContext(r)
	if err := renderDiscoverPage(w, r, page, listings); err != nil {
		support.LocalizeHTTPError(w, r, http.StatusInternalServerError, "error.http.web_handler_unavailable")
	}
}

// HandleDiscoverCampaign renders /discover/campaigns/:campaignID.
func HandleDiscoverCampaign(deps DiscoveryHandlers, w http.ResponseWriter, r *http.Request) {
	getCampaign := deps.GetCampaignListing
	pageContext := deps.PageContext
	renderError := deps.RenderErrorPage

	if getCampaign == nil || pageContext == nil || renderError == nil {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		support.LocalizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}

	campaignID := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/discover/campaigns/"))
	if campaignID == "" || strings.Contains(campaignID, "/") {
		http.NotFound(w, r)
		return
	}

	resp, err := getCampaign(r.Context(), &listingv1.GetCampaignListingRequest{CampaignId: campaignID})
	if err != nil {
		if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.NotFound {
			http.NotFound(w, r)
			return
		}
		renderError(w, r, http.StatusBadGateway, "Discovery unavailable", "failed to load campaign discovery card")
		return
	}
	listing := resp.GetListing()
	if listing == nil {
		http.NotFound(w, r)
		return
	}

	page := pageContext(r)
	if err := renderDiscoverCampaignPage(w, r, page, listingCardFromProto(listing)); err != nil {
		support.LocalizeHTTPError(w, r, http.StatusInternalServerError, "error.http.web_handler_unavailable")
	}
}

func listingCardsFromProto(in []*listingv1.CampaignListing) []publicListingCard {
	listings := make([]publicListingCard, 0, len(in))
	for _, listing := range in {
		if listing == nil {
			continue
		}
		listings = append(listings, listingCardFromProto(listing))
	}
	return listings
}

func listingCardFromProto(listing *listingv1.CampaignListing) publicListingCard {
	if listing == nil {
		return publicListingCard{}
	}
	return publicListingCard{
		CampaignID:       strings.TrimSpace(listing.GetCampaignId()),
		Title:            strings.TrimSpace(listing.GetTitle()),
		Description:      strings.TrimSpace(listing.GetDescription()),
		ExpectedDuration: strings.TrimSpace(listing.GetExpectedDurationLabel()),
	}
}

func renderDiscoverPage(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, listings []publicListingCard) error {
	return renderPublicShellPage(w, r, page, "Discover Campaigns", templ.ComponentFunc(func(_ context.Context, out io.Writer) error {
		escape := template.HTMLEscapeString
		if _, err := io.WriteString(out, `<main class="landing-shell"><section class="landing-hero"><h1>Discover Campaigns</h1><p class="hero-tagline">Browse public starter campaigns and fork one into your workspace.</p></section><section class="landing-hero"><div class="space-y-4">`); err != nil {
			return err
		}
		if len(listings) == 0 {
			if _, err := io.WriteString(out, `<p>No campaigns are published yet.</p>`); err != nil {
				return err
			}
		}
		for _, listing := range listings {
			detailLink := routepath.DiscoverCampaign(listing.CampaignID)
			if _, err := fmt.Fprintf(out, `<article class="card bg-base-200"><div class="card-body"><h2 class="card-title"><a href="%s">%s</a></h2>`, escape(detailLink), escape(listing.Title)); err != nil {
				return err
			}
			if listing.Description != "" {
				if _, err := fmt.Fprintf(out, `<p>%s</p>`, escape(listing.Description)); err != nil {
					return err
				}
			}
			if listing.ExpectedDuration != "" {
				if _, err := fmt.Fprintf(out, `<p><strong>Expected duration:</strong> %s</p>`, escape(listing.ExpectedDuration)); err != nil {
					return err
				}
			}
			if _, err := io.WriteString(out, `</div></article>`); err != nil {
				return err
			}
		}
		_, err := io.WriteString(out, `</div></section></main>`)
		return err
	}))
}

func renderDiscoverCampaignPage(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, listing publicListingCard) error {
	title := listing.Title
	if title == "" {
		title = "Campaign Listing"
	}
	return renderPublicShellPage(w, r, page, title, templ.ComponentFunc(func(_ context.Context, out io.Writer) error {
		escape := template.HTMLEscapeString
		if _, err := fmt.Fprintf(out, `<main class="landing-shell"><section class="landing-hero"><p><a href="%s">Back to discover</a></p><h1>%s</h1>`, routepath.Discover, escape(title)); err != nil {
			return err
		}
		if listing.Description != "" {
			if _, err := fmt.Fprintf(out, `<p class="hero-tagline">%s</p>`, escape(listing.Description)); err != nil {
				return err
			}
		}
		if listing.ExpectedDuration != "" {
			if _, err := fmt.Fprintf(out, `<p><strong>Expected duration:</strong> %s</p>`, escape(listing.ExpectedDuration)); err != nil {
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
