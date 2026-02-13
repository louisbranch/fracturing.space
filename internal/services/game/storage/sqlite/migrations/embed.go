package migrations

import "embed"

//go:embed events/*.sql
var EventsFS embed.FS

//go:embed projections/*.sql
var ProjectionsFS embed.FS
