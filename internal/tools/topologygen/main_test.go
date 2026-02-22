package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestParseCatalogAndRenderOutputs(t *testing.T) {
	catalogJSON := []byte(`{
  "version": 1,
  "services": [
    { "name": "web", "http_port": 8080, "public_routes": [{ "host_prefix": "", "http_port": 8080 }] },
    { "name": "auth", "grpc_port": 8083, "http_port": 8084, "public_routes": [{ "host_prefix": "auth", "http_port": 8084 }] },
    { "name": "game", "grpc_port": 8082 }
  ]
}`)

	catalog, err := parseCatalog(catalogJSON)
	if err != nil {
		t.Fatalf("parse catalog: %v", err)
	}
	if err := validateCatalog(catalog); err != nil {
		t.Fatalf("validate catalog: %v", err)
	}

	caddyRoutes, err := renderCaddyRoutes(catalog)
	if err != nil {
		t.Fatalf("render caddy routes: %v", err)
	}
	for _, want := range []string{
		"{$FRACTURING_SPACE_CADDY_SITE_PREFIX}auth.{$FRACTURING_SPACE_DOMAIN}",
		"reverse_proxy auth:8084",
		"{$FRACTURING_SPACE_CADDY_SITE_PREFIX}{$FRACTURING_SPACE_DOMAIN}",
		"reverse_proxy web:8080",
	} {
		if !strings.Contains(caddyRoutes, want) {
			t.Fatalf("expected caddy output to contain %q", want)
		}
	}

	composeFragment, err := renderComposeDiscovery(catalog)
	if err != nil {
		t.Fatalf("render compose discovery: %v", err)
	}
	for _, want := range []string{
		"x-fracturing-space-discovery:",
		"game_grpc_addr: game:8082",
		"auth_grpc_addr: auth:8083",
		"auth_http_addr: auth:8084",
	} {
		if !strings.Contains(composeFragment, want) {
			t.Fatalf("expected compose output to contain %q", want)
		}
	}

	composeFragment2, err := renderComposeDiscovery(catalog)
	if err != nil {
		t.Fatalf("render compose discovery second pass: %v", err)
	}
	if composeFragment != composeFragment2 {
		t.Fatal("expected compose output to be deterministic")
	}
}

func TestValidateCatalogRejectsDuplicateServiceNames(t *testing.T) {
	catalog, err := parseCatalog([]byte(`{
  "version": 1,
  "services": [
    { "name": "game", "grpc_port": 8082 },
    { "name": "game", "grpc_port": 19000 }
  ]
}`))
	if err != nil {
		t.Fatalf("parse catalog: %v", err)
	}
	if err := validateCatalog(catalog); err == nil {
		t.Fatal("expected duplicate service name validation error")
	}
}

func TestValidateCatalogRejectsDuplicatePublicHosts(t *testing.T) {
	catalog, err := parseCatalog([]byte(`{
  "version": 1,
  "services": [
    { "name": "auth", "http_port": 8084, "public_routes": [{ "host_prefix": "auth", "http_port": 8084 }] },
    { "name": "oauth", "http_port": 9000, "public_routes": [{ "host_prefix": "auth", "http_port": 9000 }] }
  ]
}`))
	if err != nil {
		t.Fatalf("parse catalog: %v", err)
	}
	if err := validateCatalog(catalog); err == nil {
		t.Fatal("expected duplicate public host validation error")
	}
}

func TestParseCatalogRejectsTrailingJSON(t *testing.T) {
	_, err := parseCatalog([]byte(`{
  "version": 1,
  "services": [
    { "name": "web", "http_port": 8080 }
  ]
}{"unexpected":true}`))
	if err == nil {
		t.Fatal("expected trailing content validation error")
	}
}

func TestValidateCatalogRejectsCaseInsensitiveDuplicatePublicHosts(t *testing.T) {
	catalog, err := parseCatalog([]byte(`{
  "version": 1,
  "services": [
    { "name": "auth", "http_port": 8084, "public_routes": [{ "host_prefix": "Auth", "http_port": 8084 }] },
    { "name": "oauth", "http_port": 9000, "public_routes": [{ "host_prefix": "auth", "http_port": 9000 }] }
  ]
}`))
	if err != nil {
		t.Fatalf("parse catalog: %v", err)
	}
	if err := validateCatalog(catalog); err == nil {
		t.Fatal("expected case-insensitive duplicate public host validation error")
	}
}

func TestValidateCatalogRejectsServiceNameNormalizationCollision(t *testing.T) {
	catalog, err := parseCatalog([]byte(`{
  "version": 1,
  "services": [
    { "name": "event-bus", "grpc_port": 8090 },
    { "name": "event_bus", "grpc_port": 8091 }
  ]
}`))
	if err != nil {
		t.Fatalf("parse catalog: %v", err)
	}
	if err := validateCatalog(catalog); err == nil {
		t.Fatal("expected service-name normalization collision validation error")
	}
}

func TestBootstrapUsesGeneratedComposeDiscoveryFile(t *testing.T) {
	data := mustReadRepoFile(t, "scripts", "bootstrap.sh")
	if !strings.Contains(data, "topology/generated/docker-compose.discovery.generated.yml") {
		t.Fatal("expected bootstrap.sh to include topology/generated/docker-compose.discovery.generated.yml in compose invocation")
	}
}

func mustReadRepoFile(t *testing.T, parts ...string) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve caller path")
	}

	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
	path := filepath.Join(append([]string{root}, parts...)...)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
