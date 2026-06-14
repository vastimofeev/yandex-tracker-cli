package auth

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestLoginWithBrowserExchangesAuthorizationCode(t *testing.T) {
	t.Parallel()

	var tokenForm url.Values
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm() error = %v", err)
		}
		tokenForm = r.Form
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"access_token":"token-123","refresh_token":"refresh-123","expires_in":3600,"token_type":"bearer"}`)
	}))
	defer tokenServer.Close()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	redirectURI := "http://" + listener.Addr().String() + "/callback"
	_ = listener.Close()

	authURLCh := make(chan string, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resultCh := make(chan *BrowserLoginResult, 1)
	errCh := make(chan error, 1)
	go func() {
		result, err := LoginWithBrowser(ctx, BrowserLoginOptions{
			ClientID:     "client-123",
			RedirectURI:  redirectURI,
			OpenBrowser:  false,
			Timeout:      5 * time.Second,
			TokenURL:     tokenServer.URL,
			AuthorizeURL: "https://oauth.example.test/authorize",
			OnAuthorizeURL: func(authURL string) {
				authURLCh <- authURL
			},
		})
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- result
	}()

	authURL := <-authURLCh
	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("Parse(authURL) error = %v", err)
	}
	state := parsed.Query().Get("state")
	if state == "" {
		t.Fatal("expected state in auth URL")
	}
	if parsed.Query().Get("code_challenge") == "" {
		t.Fatal("expected code_challenge in auth URL")
	}

	callbackURL := redirectURI + "?code=code-123&state=" + url.QueryEscape(state)
	resp, err := http.Get(callbackURL)
	if err != nil {
		t.Fatalf("callback GET error = %v", err)
	}
	resp.Body.Close()

	select {
	case err := <-errCh:
		t.Fatalf("LoginWithBrowser() error = %v", err)
	case result := <-resultCh:
		if result.AccessToken != "token-123" {
			t.Fatalf("AccessToken = %q, want token-123", result.AccessToken)
		}
		if tokenForm.Get("grant_type") != "authorization_code" {
			t.Fatalf("grant_type = %q, want authorization_code", tokenForm.Get("grant_type"))
		}
		if tokenForm.Get("code") != "code-123" {
			t.Fatalf("code = %q, want code-123", tokenForm.Get("code"))
		}
		if tokenForm.Get("client_id") != "client-123" {
			t.Fatalf("client_id = %q, want client-123", tokenForm.Get("client_id"))
		}
		if tokenForm.Get("redirect_uri") != redirectURI {
			t.Fatalf("redirect_uri = %q, want %q", tokenForm.Get("redirect_uri"), redirectURI)
		}
		if tokenForm.Get("code_verifier") == "" {
			t.Fatal("expected code_verifier in token exchange")
		}
	}
}

func TestLoginWithBrowserRejectsNonLocalRedirect(t *testing.T) {
	t.Parallel()

	_, err := LoginWithBrowser(context.Background(), BrowserLoginOptions{
		ClientID:    "client-123",
		RedirectURI: "https://example.com/callback",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "localhost") {
		t.Fatalf("error = %q, want localhost validation", err.Error())
	}
}

func TestFetchUserInfoUsesOAuthToken(t *testing.T) {
	t.Parallel()

	var authHeader string
	userInfoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		if r.URL.Query().Get("format") != "json" {
			t.Fatalf("format = %q, want json", r.URL.Query().Get("format"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"login":"user","default_email":"user@loov.team","emails":["user@loov.team"]}`)
	}))
	defer userInfoServer.Close()

	info, err := fetchUserInfo(context.Background(), "token-123", userInfoServer.URL, userInfoServer.Client())
	if err != nil {
		t.Fatalf("fetchUserInfo() error = %v", err)
	}
	if authHeader != "OAuth token-123" {
		t.Fatalf("Authorization = %q, want OAuth token-123", authHeader)
	}
	if info.DefaultEmail != "user@loov.team" {
		t.Fatalf("DefaultEmail = %q, want user@loov.team", info.DefaultEmail)
	}
}
