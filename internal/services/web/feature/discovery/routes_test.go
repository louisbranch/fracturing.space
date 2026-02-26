package discovery

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeDiscoveryService struct {
	discoverCalled         bool
	discoverCampaignCalled bool
}

func (f *fakeDiscoveryService) HandleDiscover(http.ResponseWriter, *http.Request) {
	f.discoverCalled = true
}

func (f *fakeDiscoveryService) HandleDiscoverCampaign(http.ResponseWriter, *http.Request) {
	f.discoverCampaignCalled = true
}

func TestRegisterRoutes(t *testing.T) {
	t.Parallel()

	svc := &fakeDiscoveryService{}
	mux := http.NewServeMux()
	RegisterRoutes(mux, svc)

	discoverReq := httptest.NewRequest(http.MethodGet, "/discover", nil)
	discoverResp := httptest.NewRecorder()
	mux.ServeHTTP(discoverResp, discoverReq)
	if discoverResp.Code != http.StatusOK {
		t.Fatalf("discover status = %d, want %d", discoverResp.Code, http.StatusOK)
	}
	if !svc.discoverCalled {
		t.Fatal("expected discover handler to be called")
	}

	detailReq := httptest.NewRequest(http.MethodGet, "/discover/campaigns/camp-1", nil)
	detailResp := httptest.NewRecorder()
	mux.ServeHTTP(detailResp, detailReq)
	if detailResp.Code != http.StatusOK {
		t.Fatalf("discover campaign status = %d, want %d", detailResp.Code, http.StatusOK)
	}
	if !svc.discoverCampaignCalled {
		t.Fatal("expected discover campaign handler to be called")
	}
}
