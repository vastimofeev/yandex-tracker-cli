package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/vastimofeev/yandex-tracker-cli/internal/config"
	"github.com/zalando/go-keyring"
)

type mockKeyring struct {
	secret string
	getErr error
	setErr error
	delErr error
}

func (m *mockKeyring) Get(service, user string) (string, error) {
	if m.getErr != nil {
		return "", m.getErr
	}
	return m.secret, nil
}

func (m *mockKeyring) Set(service, user, secret string) error {
	if m.setErr != nil {
		return m.setErr
	}
	m.secret = secret
	return nil
}

func (m *mockKeyring) Delete(service, user string) error {
	if m.delErr != nil {
		return m.delErr
	}
	m.secret = ""
	return nil
}

func TestKeyringStoreRoundTrip(t *testing.T) {
	t.Parallel()

	mock := &mockKeyring{}
	store := &KeyringStore{service: "svc", user: "usr", client: mock}
	cfg := config.StoredConfig{
		BaseURL: "https://api.example.test/v3",
		Auth: config.AuthContext{
			Token:     "token",
			TokenType: "OAuth",
			OrgID:     "org",
			OrgHeader: "X-Org-ID",
		},
	}

	if err := store.Save(context.Background(), cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Auth.Token != "token" {
		t.Fatalf("Token = %q, want token", loaded.Auth.Token)
	}
	if loaded.AuthStore != "keyring" {
		t.Fatalf("AuthStore = %q, want keyring", loaded.AuthStore)
	}
}

func TestAutoStoreFallsBackToFileWhenKeyringUnavailable(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fileStore := NewFileStore(dir + "/config.json")
	mock := &mockKeyring{
		setErr: keyring.ErrUnsupportedPlatform,
		getErr: keyring.ErrUnsupportedPlatform,
	}
	auto := NewAutoStore(
		&KeyringStore{service: "svc", user: "usr", client: mock},
		fileStore,
	)

	cfg := config.StoredConfig{
		BaseURL:   "https://api.example.test/v3",
		AuthStore: "auto",
		Auth: config.AuthContext{
			Token:     "fallback-token",
			TokenType: "OAuth",
			OrgID:     "org",
			OrgHeader: "X-Org-ID",
		},
	}

	if err := auto.Save(context.Background(), cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if auto.Kind() != "file" {
		t.Fatalf("Kind() = %q, want file", auto.Kind())
	}

	loaded, err := auto.Load(context.Background())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Auth.Token != "fallback-token" {
		t.Fatalf("Token = %q, want fallback-token", loaded.Auth.Token)
	}
}

func TestAutoStoreReturnsPrimaryErrorsWhenNotUnavailable(t *testing.T) {
	t.Parallel()

	auto := NewAutoStore(
		&KeyringStore{service: "svc", user: "usr", client: &mockKeyring{setErr: errors.New("access denied")}},
		NewFileStore(t.TempDir()+"/config.json"),
	)

	err := auto.Save(context.Background(), config.StoredConfig{})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "access denied" {
		t.Fatalf("error = %q, want access denied", err.Error())
	}
}
