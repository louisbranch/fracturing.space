// Package discovery centralizes internal service-discovery conventions.
package discovery

import (
	"strconv"
	"strings"
)

const (
	// ServiceAdmin is the admin dashboard service identity.
	ServiceAdmin = "admin"
	// ServiceAI is the AI gRPC service identity.
	ServiceAI = "ai"
	// ServiceSocial is the social gRPC service identity.
	ServiceSocial = "social"
	// ServiceListing is the listing gRPC service identity.
	ServiceListing = "listing"
	// ServiceAuth is the auth service identity.
	ServiceAuth = "auth"
	// ServiceChat is the chat HTTP service identity.
	ServiceChat = "chat"
	// ServiceGame is the game gRPC service identity.
	ServiceGame = "game"
	// ServiceJaeger is the jaeger HTTP service identity.
	ServiceJaeger = "jaeger"
	// ServiceMCP is the MCP HTTP service identity.
	ServiceMCP = "mcp"
	// ServiceNotifications is the notifications service identity.
	ServiceNotifications = "notifications"
	// ServiceWeb is the web login HTTP service identity.
	ServiceWeb = "web"
	// ServiceWorker is the worker gRPC service identity.
	ServiceWorker = "worker"
)

var grpcPorts = map[string]int{
	ServiceGame:          8082,
	ServiceAuth:          8083,
	ServiceSocial:        8090,
	ServiceListing:       8091,
	ServiceAI:            8087,
	ServiceNotifications: 8088,
	ServiceWorker:        8089,
}

var httpPorts = map[string]int{
	ServiceWeb:    8080,
	ServiceAdmin:  8081,
	ServiceAuth:   8084,
	ServiceMCP:    8085,
	ServiceChat:   8086,
	ServiceJaeger: 16686,
}

// DefaultGRPCAddr returns the canonical in-network gRPC address for a service.
func DefaultGRPCAddr(service string) string {
	return defaultAddr(strings.TrimSpace(service), grpcPorts)
}

// DefaultHTTPAddr returns the canonical in-network HTTP address for a service.
func DefaultHTTPAddr(service string) string {
	return defaultAddr(strings.TrimSpace(service), httpPorts)
}

// OrDefaultGRPCAddr returns value when set, otherwise the service convention.
func OrDefaultGRPCAddr(value, service string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return DefaultGRPCAddr(service)
}

// OrDefaultHTTPAddr returns value when set, otherwise the service convention.
func OrDefaultHTTPAddr(value, service string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return DefaultHTTPAddr(service)
}

// OrDefaultHTTPBaseURL returns value when set, otherwise http://<service-host:port>.
func OrDefaultHTTPBaseURL(value, service string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	addr := DefaultHTTPAddr(service)
	if addr == "" {
		return ""
	}
	return "http://" + addr
}

func defaultAddr(service string, ports map[string]int) string {
	port, ok := ports[service]
	if !ok || port <= 0 {
		return ""
	}
	return service + ":" + strconv.Itoa(port)
}
