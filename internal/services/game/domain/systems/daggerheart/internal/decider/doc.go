// Package decider implements the Daggerheart system command decision handlers.
//
// Each handler validates and transforms a command into zero or more domain
// events. The Decider type routes commands by type to the appropriate handler.
package decider
