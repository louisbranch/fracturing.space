package dashboard

import (
	"context"

	"golang.org/x/text/language"
)

// fakeGateway implements DashboardGateway for tests with configurable return
// values and call tracking.
type fakeGateway struct {
	snapshot DashboardSnapshot
	err      error
	calls    int
}

func (f *fakeGateway) LoadDashboard(context.Context, string, language.Tag) (DashboardSnapshot, error) {
	f.calls++
	if f.err != nil {
		return DashboardSnapshot{}, f.err
	}
	return f.snapshot, nil
}
