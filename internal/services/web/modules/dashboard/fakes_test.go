package dashboard

import (
	"context"

	dashboardapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/dashboard/app"
	"golang.org/x/text/language"
)

// fakeGateway implements dashboardapp.Gateway for tests with configurable return
// values and call tracking.
type fakeGateway struct {
	snapshot dashboardapp.DashboardSnapshot
	err      error
	calls    int
}

func (f *fakeGateway) LoadDashboard(context.Context, string, language.Tag) (dashboardapp.DashboardSnapshot, error) {
	f.calls++
	if f.err != nil {
		return dashboardapp.DashboardSnapshot{}, f.err
	}
	return f.snapshot, nil
}
