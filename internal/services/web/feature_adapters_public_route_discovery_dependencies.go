package web

import (
	"context"
	"net/http"

	listingv1 "github.com/louisbranch/fracturing.space/api/gen/go/listing/v1"
	discoveryfeature "github.com/louisbranch/fracturing.space/internal/services/web/feature/discovery"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
	"google.golang.org/grpc"
)

func (h *handler) publicDiscoveryRouteDependencies(w http.ResponseWriter, r *http.Request) discoveryfeature.DiscoveryHandlers {
	var listCampaignListings func(context.Context, *listingv1.ListCampaignListingsRequest, ...grpc.CallOption) (*listingv1.ListCampaignListingsResponse, error)
	if h.listingClient != nil {
		listCampaignListings = func(ctx context.Context, req *listingv1.ListCampaignListingsRequest, opts ...grpc.CallOption) (*listingv1.ListCampaignListingsResponse, error) {
			return h.listingClient.ListCampaignListings(ctx, req, opts...)
		}
	}

	var getCampaignListing func(context.Context, *listingv1.GetCampaignListingRequest, ...grpc.CallOption) (*listingv1.GetCampaignListingResponse, error)
	if h.listingClient != nil {
		getCampaignListing = func(ctx context.Context, req *listingv1.GetCampaignListingRequest, opts ...grpc.CallOption) (*listingv1.GetCampaignListingResponse, error) {
			return h.listingClient.GetCampaignListing(ctx, req, opts...)
		}
	}

	return discoveryfeature.DiscoveryHandlers{
		ListCampaignListings: listCampaignListings,
		GetCampaignListing:   getCampaignListing,
		PageContext: func(req *http.Request) webtemplates.PageContext {
			return h.pageContext(w, req)
		},
		RenderErrorPage: h.renderErrorPage,
	}
}
