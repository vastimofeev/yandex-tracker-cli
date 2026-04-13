package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const (
	DefaultBaseURL           = "https://api.tracker.yandex.net/v3"
	DefaultTokenType         = "OAuth"
	DefaultOrgHeader         = "X-Org-ID"
	DefaultAuthStore         = "keyring"
	DefaultOAuthClientID     = "40fd180c589c4ffabc99a2895f5f3395"
	DefaultOAuthRedirectURI  = "https://oauth.yandex.ru/verification_code"
	DefaultOAuthResponseType = "token"
	EnvToken                 = "YANDEX_TRACKER_TOKEN"
	EnvTokenType             = "YANDEX_TRACKER_TOKEN_TYPE"
	EnvOrgID                 = "YANDEX_TRACKER_ORG_ID"
	EnvOrgHeader             = "YANDEX_TRACKER_ORG_HEADER"
	EnvBaseURL               = "YANDEX_TRACKER_BASE_URL"
	EnvAuthStore             = "YANDEX_TRACKER_AUTH_STORE"
	EnvOAuthClientID         = "YANDEX_TRACKER_OAUTH_CLIENT_ID"
	EnvOAuthRedirectURI      = "YANDEX_TRACKER_OAUTH_REDIRECT_URI"
	EnvOAuthScope            = "YANDEX_TRACKER_OAUTH_SCOPE"
	EnvOAuthOptScope         = "YANDEX_TRACKER_OAUTH_OPTIONAL_SCOPE"
	EnvOAuthLoginHint        = "YANDEX_TRACKER_OAUTH_LOGIN_HINT"
	EnvOAuthResponseType     = "YANDEX_TRACKER_OAUTH_RESPONSE_TYPE"
	DefaultConfigSubdir      = "yandex-tracker-cli"
	DefaultConfigFile        = "config.json"
)

var errMissingHome = errors.New("could not determine user home directory")

type AuthContext struct {
	Token     string `json:"token"`
	TokenType string `json:"token_type"`
	OrgID     string `json:"org_id"`
	OrgHeader string `json:"org_header"`
}

func (a AuthContext) Complete() bool {
	return a.Token != "" && a.TokenType != "" && a.OrgID != "" && a.OrgHeader != ""
}

type StoredConfig struct {
	BaseURL   string      `json:"base_url"`
	Auth      AuthContext `json:"auth"`
	AuthStore string      `json:"auth_store,omitempty"`
	OAuth     OAuthConfig `json:"oauth,omitempty"`
}

type OAuthConfig struct {
	ClientID      string `json:"client_id,omitempty"`
	RedirectURI   string `json:"redirect_uri,omitempty"`
	Scope         string `json:"scope,omitempty"`
	OptionalScope string `json:"optional_scope,omitempty"`
	LoginHint     string `json:"login_hint,omitempty"`
	ResponseType  string `json:"response_type,omitempty"`
}

type CLIOptions struct {
	ConfigPath         string
	BaseURL            string
	Token              string
	TokenType          string
	OrgID              string
	OrgHeader          string
	AuthStore          string
	OAuthClientID      string
	OAuthRedirectURI   string
	OAuthScope         string
	OAuthOptionalScope string
	OAuthLoginHint     string
	OAuthResponseType  string
	JSON               bool
	Debug              bool
}

type RuntimeConfig struct {
	ConfigPath string
	BaseURL    string
	Auth       AuthContext
	AuthStore  string
	OAuth      OAuthConfig
	JSON       bool
	Debug      bool
}

type EnvLookup func(string) (string, bool)

func OSEnv(key string) (string, bool) {
	return os.LookupEnv(key)
}

func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", errMissingHome
	}
	return filepath.Join(home, ".config", DefaultConfigSubdir, DefaultConfigFile), nil
}

func NormalizeTokenType(v string) string {
	switch strings.TrimSpace(strings.ToLower(v)) {
	case "", "oauth":
		return DefaultTokenType
	case "bearer":
		return "Bearer"
	default:
		return v
	}
}

func NormalizeOrgHeader(v string) string {
	switch strings.TrimSpace(strings.ToLower(v)) {
	case "", "x-org-id":
		return DefaultOrgHeader
	case "x-cloud-org-id":
		return "X-Cloud-Org-ID"
	default:
		return v
	}
}

func NormalizeAuthStore(v string) string {
	switch strings.TrimSpace(strings.ToLower(v)) {
	case "", "auto":
		return DefaultAuthStore
	case "keyring":
		return "keyring"
	case "file":
		return "file"
	default:
		return v
	}
}

func NormalizeOAuthResponseType(v string) string {
	switch strings.TrimSpace(strings.ToLower(v)) {
	case "", "token":
		return DefaultOAuthResponseType
	case "code":
		return "code"
	default:
		return v
	}
}

func Resolve(opts CLIOptions, env EnvLookup, stored StoredConfig) (RuntimeConfig, error) {
	if env == nil {
		env = OSEnv
	}
	configPath := opts.ConfigPath
	if configPath == "" {
		var err error
		configPath, err = DefaultPath()
		if err != nil {
			return RuntimeConfig{}, err
		}
	}

	auth := stored.Auth
	if v := firstNonEmpty(opts.Token, envValue(env, EnvToken)); v != "" {
		auth.Token = v
	}
	if v := firstNonEmpty(opts.TokenType, envValue(env, EnvTokenType), auth.TokenType, DefaultTokenType); v != "" {
		auth.TokenType = NormalizeTokenType(v)
	}
	if v := firstNonEmpty(opts.OrgID, envValue(env, EnvOrgID), auth.OrgID); v != "" {
		auth.OrgID = v
	}
	if v := firstNonEmpty(opts.OrgHeader, envValue(env, EnvOrgHeader), auth.OrgHeader, DefaultOrgHeader); v != "" {
		auth.OrgHeader = NormalizeOrgHeader(v)
	}

	baseURL := firstNonEmpty(opts.BaseURL, envValue(env, EnvBaseURL), stored.BaseURL, DefaultBaseURL)
	authStore := firstNonEmpty(opts.AuthStore, envValue(env, EnvAuthStore), stored.AuthStore, DefaultAuthStore)
	oauth := stored.OAuth
	if v := firstNonEmpty(opts.OAuthClientID, envValue(env, EnvOAuthClientID), oauth.ClientID, DefaultOAuthClientID); v != "" {
		oauth.ClientID = v
	}
	if v := firstNonEmpty(opts.OAuthRedirectURI, envValue(env, EnvOAuthRedirectURI), oauth.RedirectURI, DefaultOAuthRedirectURI); v != "" {
		oauth.RedirectURI = v
	}
	if v := firstNonEmpty(opts.OAuthScope, envValue(env, EnvOAuthScope), oauth.Scope); v != "" {
		oauth.Scope = v
	}
	if v := firstNonEmpty(opts.OAuthOptionalScope, envValue(env, EnvOAuthOptScope), oauth.OptionalScope); v != "" {
		oauth.OptionalScope = v
	}
	if v := firstNonEmpty(opts.OAuthLoginHint, envValue(env, EnvOAuthLoginHint), oauth.LoginHint); v != "" {
		oauth.LoginHint = v
	}
	oauth.ResponseType = NormalizeOAuthResponseType(firstNonEmpty(opts.OAuthResponseType, envValue(env, EnvOAuthResponseType), oauth.ResponseType, DefaultOAuthResponseType))

	return RuntimeConfig{
		ConfigPath: configPath,
		BaseURL:    strings.TrimRight(baseURL, "/"),
		Auth:       auth,
		AuthStore:  NormalizeAuthStore(authStore),
		OAuth:      oauth,
		JSON:       opts.JSON,
		Debug:      opts.Debug,
	}, nil
}

func envValue(env EnvLookup, key string) string {
	if env == nil {
		return ""
	}
	v, ok := env(key)
	if !ok {
		return ""
	}
	return v
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
