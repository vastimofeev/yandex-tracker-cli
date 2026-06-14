package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	goruntime "runtime"
	"strings"
	"time"
)

const (
	DefaultAuthorizeURL = "https://oauth.yandex.ru/authorize"
	DefaultTokenURL     = "https://oauth.yandex.ru/token"
	DefaultUserInfoURL  = "https://login.yandex.ru/info"
	DefaultDeviceName   = "yandex-tracker-cli"
)

type BrowserLoginOptions struct {
	ClientID       string
	RedirectURI    string
	Scope          string
	OptionalScope  string
	LoginHint      string
	ResponseType   string
	ForceConfirm   bool
	OpenBrowser    bool
	Timeout        time.Duration
	AuthorizeURL   string
	TokenURL       string
	DeviceName     string
	HTTPClient     *http.Client
	OnAuthorizeURL func(string)
}

type BrowserLoginResult struct {
	AuthorizationURL string `json:"authorizationUrl,omitempty"`
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token,omitempty"`
	ExpiresIn        int    `json:"expires_in,omitempty"`
	TokenType        string `json:"token_type,omitempty"`
}

type UserInfo struct {
	ID           string   `json:"id,omitempty"`
	Login        string   `json:"login,omitempty"`
	DisplayName  string   `json:"display_name,omitempty"`
	RealName     string   `json:"real_name,omitempty"`
	DefaultEmail string   `json:"default_email,omitempty"`
	Emails       []string `json:"emails,omitempty"`
}

type callbackResult struct {
	Code  string
	State string
	Err   string
}

func LoginWithBrowser(ctx context.Context, opts BrowserLoginOptions) (*BrowserLoginResult, error) {
	if strings.TrimSpace(opts.ClientID) == "" {
		return nil, fmt.Errorf("oauth client ID is required")
	}
	if strings.TrimSpace(opts.RedirectURI) == "" {
		return nil, fmt.Errorf("oauth redirect URI is required")
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 2 * time.Minute
	}
	if opts.AuthorizeURL == "" {
		opts.AuthorizeURL = DefaultAuthorizeURL
	}
	if opts.TokenURL == "" {
		opts.TokenURL = DefaultTokenURL
	}
	if opts.DeviceName == "" {
		opts.DeviceName = DefaultDeviceName
	}
	if opts.HTTPClient == nil {
		opts.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	}
	if opts.ResponseType == "" {
		opts.ResponseType = "code"
	}
	if opts.ResponseType != "code" {
		return nil, fmt.Errorf("browser code flow requires response_type=code")
	}

	redirectURL, err := url.Parse(opts.RedirectURI)
	if err != nil {
		return nil, fmt.Errorf("parse redirect URI: %w", err)
	}
	if !isLocalRedirect(redirectURL) {
		return nil, fmt.Errorf("redirect URI must point to localhost or 127.0.0.1")
	}

	state, err := randomToken(24)
	if err != nil {
		return nil, err
	}
	verifier, err := randomToken(48)
	if err != nil {
		return nil, err
	}
	challenge := pkceChallenge(verifier)
	deviceID, err := randomToken(16)
	if err != nil {
		return nil, err
	}

	listener, err := net.Listen("tcp", redirectURL.Host)
	if err != nil {
		return nil, fmt.Errorf("listen on redirect URI: %w", err)
	}
	defer listener.Close()

	callbackCh := make(chan callbackResult, 1)
	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != redirectURL.Path {
				http.NotFound(w, r)
				return
			}
			result := callbackResult{
				Code:  r.URL.Query().Get("code"),
				State: r.URL.Query().Get("state"),
				Err:   r.URL.Query().Get("error"),
			}
			message := "Authorization received. You can return to the CLI."
			if result.Err != "" {
				message = "Authorization failed. You can return to the CLI."
			}
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = io.WriteString(w, message)
			select {
			case callbackCh <- result:
			default:
			}
		}),
	}

	go func() {
		_ = server.Serve(listener)
	}()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	authURL := buildAuthorizeURL(opts, redirectURL.String(), state, challenge, deviceID)
	if opts.OnAuthorizeURL != nil {
		opts.OnAuthorizeURL(authURL)
	}
	if opts.OpenBrowser {
		if err := openBrowser(authURL); err != nil {
			return nil, err
		}
	}

	waitCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	var callback callbackResult
	select {
	case <-waitCtx.Done():
		return nil, fmt.Errorf("oauth browser login timed out")
	case callback = <-callbackCh:
	}

	if callback.Err != "" {
		return nil, fmt.Errorf("oauth authorization failed: %s", callback.Err)
	}
	if callback.State != state {
		return nil, fmt.Errorf("oauth state mismatch")
	}
	if callback.Code == "" {
		return nil, fmt.Errorf("oauth callback did not return a code")
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", callback.Code)
	form.Set("client_id", opts.ClientID)
	form.Set("code_verifier", verifier)
	form.Set("device_id", deviceID)
	form.Set("device_name", opts.DeviceName)
	form.Set("redirect_uri", redirectURL.String())

	req, err := http.NewRequestWithContext(waitCtx, http.MethodPost, opts.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := opts.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("oauth token exchange failed: %s", strings.TrimSpace(string(body)))
	}

	var tokenResponse struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
	}
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return nil, err
	}
	if tokenResponse.AccessToken == "" {
		return nil, fmt.Errorf("oauth token exchange returned no access token")
	}

	return &BrowserLoginResult{
		AuthorizationURL: authURL,
		AccessToken:      tokenResponse.AccessToken,
		RefreshToken:     tokenResponse.RefreshToken,
		ExpiresIn:        tokenResponse.ExpiresIn,
		TokenType:        tokenResponse.TokenType,
	}, nil
}

func buildAuthorizeURL(opts BrowserLoginOptions, redirectURI, state, challenge, deviceID string) string {
	values := url.Values{}
	values.Set("response_type", opts.ResponseType)
	values.Set("client_id", opts.ClientID)
	if redirectURI != "" {
		values.Set("redirect_uri", redirectURI)
	}
	if state != "" {
		values.Set("state", state)
	}
	if challenge != "" {
		values.Set("code_challenge", challenge)
		values.Set("code_challenge_method", "S256")
	}
	if deviceID != "" {
		values.Set("device_id", deviceID)
		values.Set("device_name", opts.DeviceName)
	}
	if opts.Scope != "" {
		values.Set("scope", opts.Scope)
	}
	if opts.OptionalScope != "" {
		values.Set("optional_scope", opts.OptionalScope)
	}
	if opts.LoginHint != "" {
		values.Set("login_hint", opts.LoginHint)
	}
	if opts.ForceConfirm {
		values.Set("force_confirm", "yes")
	}
	return opts.AuthorizeURL + "?" + values.Encode()
}

func BuildTokenAuthorizeURL(opts BrowserLoginOptions) (string, error) {
	if strings.TrimSpace(opts.ClientID) == "" {
		return "", fmt.Errorf("oauth client ID is required")
	}
	if opts.AuthorizeURL == "" {
		opts.AuthorizeURL = DefaultAuthorizeURL
	}
	opts.ResponseType = "token"
	return buildAuthorizeURL(opts, opts.RedirectURI, "", "", ""), nil
}

func FetchUserInfo(ctx context.Context, token string, httpClient *http.Client) (*UserInfo, error) {
	return fetchUserInfo(ctx, token, DefaultUserInfoURL, httpClient)
}

func fetchUserInfo(ctx context.Context, token, rawURL string, httpClient *http.Client) (*UserInfo, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, fmt.Errorf("oauth token is required")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	infoURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	query := infoURL.Query()
	query.Set("format", "json")
	infoURL.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, infoURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "OAuth "+token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("oauth user info request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var info UserInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, err
	}
	if info.Login == "" && info.DefaultEmail == "" && len(info.Emails) == 0 {
		return nil, fmt.Errorf("oauth user info response did not include login or email")
	}
	return &info, nil
}

func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func randomToken(bytesLen int) (string, error) {
	buf := make([]byte, bytesLen)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func isLocalRedirect(u *url.URL) bool {
	host := strings.ToLower(u.Hostname())
	return host == "127.0.0.1" || host == "localhost"
}

func openBrowser(target string) error {
	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	case "darwin":
		cmd = exec.Command("open", target)
	default:
		cmd = exec.Command("xdg-open", target)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open browser: %w", err)
	}
	return nil
}

func OpenBrowserURL(target string) error {
	return openBrowser(target)
}
