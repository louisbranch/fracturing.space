package discovery

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDefaultGRPCAddr(t *testing.T) {
	cases := map[string]string{
		ServiceGame:          "game:8082",
		ServiceAuth:          "auth:8083",
		ServiceSocial:        "social:8090",
		ServiceListing:       "listing:8091",
		ServiceAI:            "ai:8087",
		ServiceNotifications: "notifications:8088",
	}
	for service, want := range cases {
		if got := DefaultGRPCAddr(service); got != want {
			t.Fatalf("DefaultGRPCAddr(%q) = %q, want %q", service, got, want)
		}
	}
}

func TestDefaultHTTPAddr(t *testing.T) {
	cases := map[string]string{
		ServiceAuth:   "auth:8084",
		ServiceWeb:    "web:8080",
		ServiceAdmin:  "admin:8081",
		ServiceMCP:    "mcp:8085",
		ServiceChat:   "chat:8086",
		ServiceJaeger: "jaeger:16686",
	}
	for service, want := range cases {
		if got := DefaultHTTPAddr(service); got != want {
			t.Fatalf("DefaultHTTPAddr(%q) = %q, want %q", service, got, want)
		}
	}
}

func TestOrDefaultGRPCAddr(t *testing.T) {
	if got := OrDefaultGRPCAddr(" custom:9000 ", ServiceAuth); got != "custom:9000" {
		t.Fatalf("expected explicit grpc addr to win, got %q", got)
	}
	if got := OrDefaultGRPCAddr("", ServiceAuth); got != "auth:8083" {
		t.Fatalf("expected default grpc addr, got %q", got)
	}
}

func TestOrDefaultHTTPBaseURL(t *testing.T) {
	if got := OrDefaultHTTPBaseURL(" https://issuer.example.com ", ServiceAuth); got != "https://issuer.example.com" {
		t.Fatalf("expected explicit base url to win, got %q", got)
	}
	if got := OrDefaultHTTPBaseURL("", ServiceAuth); got != "http://auth:8084" {
		t.Fatalf("expected default auth base url, got %q", got)
	}
}

func TestDiscoveryDefaultsMatchTopologyCatalog(t *testing.T) {
	grpcFromCatalog, httpFromCatalog := readTopologyPorts(t)

	for service, port := range grpcFromCatalog {
		want := fmt.Sprintf("%s:%d", service, port)
		if got := DefaultGRPCAddr(service); got != want {
			t.Fatalf("catalog grpc default mismatch for %q: got %q, want %q", service, got, want)
		}
	}
	for service, port := range httpFromCatalog {
		want := fmt.Sprintf("%s:%d", service, port)
		if got := DefaultHTTPAddr(service); got != want {
			t.Fatalf("catalog http default mismatch for %q: got %q, want %q", service, got, want)
		}
	}

	for service := range grpcPorts {
		if _, ok := grpcFromCatalog[service]; !ok {
			t.Fatalf("grpc defaults include service %q not present in topology catalog", service)
		}
	}
	for service := range httpPorts {
		if _, ok := httpFromCatalog[service]; !ok {
			t.Fatalf("http defaults include service %q not present in topology catalog", service)
		}
	}
}

func readTopologyPorts(t *testing.T) (map[string]int, map[string]int) {
	t.Helper()

	type topologyService struct {
		Name     string `json:"name"`
		GRPCPort int    `json:"grpc_port"`
		HTTPPort int    `json:"http_port"`
	}
	type topologyCatalog struct {
		Services []topologyService `json:"services"`
	}

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve caller path")
	}

	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
	data, err := os.ReadFile(filepath.Join(root, "topology", "services.json"))
	if err != nil {
		t.Fatalf("read topology/services.json: %v", err)
	}

	var parsed topologyCatalog
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("parse topology/services.json: %v", err)
	}

	grpcPortsFromCatalog := make(map[string]int, len(parsed.Services))
	httpPortsFromCatalog := make(map[string]int, len(parsed.Services))
	for _, svc := range parsed.Services {
		if svc.GRPCPort > 0 {
			grpcPortsFromCatalog[svc.Name] = svc.GRPCPort
		}
		if svc.HTTPPort > 0 {
			httpPortsFromCatalog[svc.Name] = svc.HTTPPort
		}
	}
	return grpcPortsFromCatalog, httpPortsFromCatalog
}
