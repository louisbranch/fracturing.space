// Package server is the AI service composition root and runtime lifecycle.
//
// It wires the handler roots, provider adapters, prompt render policy, storage,
// and cross-service clients into the live process so contributors can find the
// concrete runtime shape in one package without recovering it from tests.
package server
