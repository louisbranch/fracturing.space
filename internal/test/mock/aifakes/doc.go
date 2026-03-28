// Package aifakes provides capability-specific in-memory seams for AI service
// tests.
//
// Each fake is intentionally scoped to one repository or helper boundary so
// tests compose only the seams they actually depend on. When one test needs
// workflow-specific counters or bespoke behavior, prefer a package-local fake
// in that test package over growing aifakes into a second application layer.
package aifakes
