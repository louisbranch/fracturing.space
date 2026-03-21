package app

import (
	"context"

	"golang.org/x/text/language"
)

// Gateway loads dashboard snapshot data for one user.
type Gateway interface {
	LoadDashboard(context.Context, string, language.Tag) (DashboardSnapshot, error)
}

// Service exposes dashboard orchestration methods used by transport handlers.
type Service interface {
	LoadDashboard(context.Context, string, language.Tag) (DashboardView, error)
}
