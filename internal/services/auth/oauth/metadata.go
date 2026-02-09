package oauth

import (
	"encoding/json"
	"net/http"
	"strings"
)

// AuthorizationServerMetadata represents OAuth 2.0 Authorization Server Metadata.
type AuthorizationServerMetadata struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	IntrospectionEndpoint             string   `json:"introspection_endpoint"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	GrantTypesSupported               []string `json:"grant_types_supported"`
	CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
}

func (s *Server) handleMetadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	issuer := strings.TrimRight(s.config.Issuer, "/")
	if issuer == "" {
		issuer = issuerFromRequest(r)
	}

	metadata := AuthorizationServerMetadata{
		Issuer:                            issuer,
		AuthorizationEndpoint:             issuer + "/authorize",
		TokenEndpoint:                     issuer + "/token",
		IntrospectionEndpoint:             issuer + "/introspect",
		ResponseTypesSupported:            []string{"code"},
		GrantTypesSupported:               []string{"authorization_code"},
		CodeChallengeMethodsSupported:     []string{"S256"},
		TokenEndpointAuthMethodsSupported: tokenAuthMethodsSupported(s.config.Clients),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(metadata)
}

func tokenAuthMethodsSupported(clients []Client) []string {
	methods := []string{"none"}
	for _, client := range clients {
		if client.Secret != "" && client.TokenEndpointAuthMethod != "none" {
			methods = append(methods, "client_secret_post")
			break
		}
	}
	return methods
}

func issuerFromRequest(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwarded := r.Header.Get("X-Forwarded-Proto"); forwarded != "" {
		scheme = forwarded
	}
	host := r.Host
	return scheme + "://" + host
}
