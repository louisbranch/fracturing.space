// Package charactertransport owns protobuf mapping and Daggerheart transport
// conversions for character-facing game transport flows.
//
// Keeping character records, system profile/state mapping, and Daggerheart
// condition/life-state conversions here gives character and snapshot handlers a
// character-owned import boundary instead of reusing root-package helpers.
package charactertransport
