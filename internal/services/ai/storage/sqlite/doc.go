// Package sqlite provides the SQLite-backed AI persistence adapter.
//
// One runtime root owns database opening, migrations, and low-level helpers,
// while aggregate behavior is kept in family-local files and tests so
// credential, agent, grant, access-request, audit-event, and artifact
// persistence remain discoverable to contributors.
package sqlite
