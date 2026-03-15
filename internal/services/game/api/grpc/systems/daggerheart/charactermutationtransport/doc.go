// Package charactermutationtransport owns the Daggerheart character-scoped
// progression and inventory write endpoints.
//
// This slice groups the transport behavior that shares one character-targeted
// command pattern: campaign mutation validation, Daggerheart system checks,
// character profile existence checks, payload shaping, and post-write profile
// reloads.
package charactermutationtransport
