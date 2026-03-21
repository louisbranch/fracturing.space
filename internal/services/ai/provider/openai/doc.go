// Package openai provides the concrete OpenAI adapters used by the AI service.
//
// Responsibility is split by boundary within this package: OAuth token flows,
// Responses API translation, model listing, and strict-schema shaping each
// live in dedicated files so provider-specific behavior does not collapse back
// into one omnibus adapter.
package openai
