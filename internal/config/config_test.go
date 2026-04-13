package config

import "testing"

func TestResolvePrecedence(t *testing.T) {
	t.Parallel()

	stored := StoredConfig{
		BaseURL: "https://stored.example/v3",
		Auth: AuthContext{
			Token:     "stored-token",
			TokenType: "Bearer",
			OrgID:     "stored-org",
			OrgHeader: "X-Cloud-Org-ID",
		},
	}
	env := func(key string) (string, bool) {
		values := map[string]string{
			EnvToken:            "env-token",
			EnvTokenType:        "oauth",
			EnvOrgID:            "env-org",
			EnvOrgHeader:        "x-org-id",
			EnvBaseURL:          "https://env.example/v3",
			EnvAuthStore:        "file",
			EnvOAuthClientID:    "env-client",
			EnvOAuthRedirectURI: "http://127.0.0.1:8485/callback",
		}
		v, ok := values[key]
		return v, ok
	}

	cfg, err := Resolve(CLIOptions{
		ConfigPath:       "test-config.json",
		BaseURL:          "https://flag.example/v3",
		Token:            "flag-token",
		TokenType:        "bearer",
		OrgID:            "flag-org",
		OrgHeader:        "x-cloud-org-id",
		AuthStore:        "keyring",
		OAuthClientID:    "flag-client",
		OAuthRedirectURI: "http://127.0.0.1:9999/callback",
		JSON:             true,
		Debug:            true,
	}, env, stored)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if cfg.BaseURL != "https://flag.example/v3" {
		t.Fatalf("BaseURL = %q, want %q", cfg.BaseURL, "https://flag.example/v3")
	}
	if cfg.Auth.Token != "flag-token" {
		t.Fatalf("Token = %q, want %q", cfg.Auth.Token, "flag-token")
	}
	if cfg.Auth.TokenType != "Bearer" {
		t.Fatalf("TokenType = %q, want Bearer", cfg.Auth.TokenType)
	}
	if cfg.Auth.OrgID != "flag-org" {
		t.Fatalf("OrgID = %q, want flag-org", cfg.Auth.OrgID)
	}
	if cfg.Auth.OrgHeader != "X-Cloud-Org-ID" {
		t.Fatalf("OrgHeader = %q, want X-Cloud-Org-ID", cfg.Auth.OrgHeader)
	}
	if cfg.AuthStore != "keyring" {
		t.Fatalf("AuthStore = %q, want keyring", cfg.AuthStore)
	}
	if cfg.OAuth.ClientID != "flag-client" {
		t.Fatalf("OAuth.ClientID = %q, want flag-client", cfg.OAuth.ClientID)
	}
	if cfg.OAuth.RedirectURI != "http://127.0.0.1:9999/callback" {
		t.Fatalf("OAuth.RedirectURI = %q", cfg.OAuth.RedirectURI)
	}
	if !cfg.JSON || !cfg.Debug {
		t.Fatalf("expected JSON and Debug to be true")
	}
}

func TestResolveUsesEnvThenStoredDefaults(t *testing.T) {
	t.Parallel()

	env := func(key string) (string, bool) {
		values := map[string]string{
			EnvToken:     "env-token",
			EnvTokenType: "oauth",
		}
		v, ok := values[key]
		return v, ok
	}

	cfg, err := Resolve(CLIOptions{ConfigPath: "test-config.json"}, env, StoredConfig{
		Auth: AuthContext{
			OrgID:     "stored-org",
			OrgHeader: "X-Cloud-Org-ID",
		},
	})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if cfg.BaseURL != DefaultBaseURL {
		t.Fatalf("BaseURL = %q, want %q", cfg.BaseURL, DefaultBaseURL)
	}
	if cfg.Auth.Token != "env-token" {
		t.Fatalf("Token = %q, want env-token", cfg.Auth.Token)
	}
	if cfg.Auth.TokenType != "OAuth" {
		t.Fatalf("TokenType = %q, want OAuth", cfg.Auth.TokenType)
	}
	if cfg.Auth.OrgID != "stored-org" {
		t.Fatalf("OrgID = %q, want stored-org", cfg.Auth.OrgID)
	}
	if cfg.Auth.OrgHeader != "X-Cloud-Org-ID" {
		t.Fatalf("OrgHeader = %q, want X-Cloud-Org-ID", cfg.Auth.OrgHeader)
	}
	if cfg.AuthStore != DefaultAuthStore {
		t.Fatalf("AuthStore = %q, want %q", cfg.AuthStore, DefaultAuthStore)
	}
	if cfg.OAuth.ClientID != DefaultOAuthClientID {
		t.Fatalf("OAuth.ClientID = %q, want %q", cfg.OAuth.ClientID, DefaultOAuthClientID)
	}
}
