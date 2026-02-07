// Package metrics provides operational metrics collection.
//
// This package handles system observability for monitoring and alerting:
//
// # Metric Categories
//
//   - Latency: Request duration histograms by endpoint
//   - Errors: Error counts by type and endpoint
//   - Usage: API call counts, active sessions, concurrent users
//   - Resources: Memory, goroutines, connections
//
// # Integration
//
// Metrics are collected via gRPC interceptors and exposed in Prometheus format.
// The metrics endpoint can be scraped by standard monitoring infrastructure.
//
// # gRPC Interceptor
//
// The interceptor automatically records:
//   - Request count by method
//   - Request latency by method
//   - Error count by method and code
//
// # Future Enhancements
//
// Planned features:
//   - Custom business metrics (rolls per session, etc.)
//   - Distributed tracing integration
//   - Health check aggregation
package metrics
