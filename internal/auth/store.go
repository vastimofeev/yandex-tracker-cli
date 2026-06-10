package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vastimofeev/yandex-tracker-cli/internal/config"
	"github.com/zalando/go-keyring"
)

var ErrKeyringNotImplemented = errors.New("keyring store is not implemented")

type TokenStore interface {
	Load(context.Context) (config.StoredConfig, error)
	Save(context.Context, config.StoredConfig) error
	Clear(context.Context) error
}

type StoreDescriptor interface {
	Kind() string
}

type FileStore struct {
	path string
}

func NewFileStore(path string) *FileStore {
	return &FileStore{path: path}
}

func (s *FileStore) Load(context.Context) (config.StoredConfig, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return config.StoredConfig{}, nil
		}
		return config.StoredConfig{}, err
	}
	var cfg config.StoredConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return config.StoredConfig{}, err
	}
	return cfg, nil
}

func (s *FileStore) Save(_ context.Context, cfg config.StoredConfig) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o600)
}

func (s *FileStore) Clear(context.Context) error {
	err := os.Remove(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func (s *FileStore) Kind() string {
	return "file"
}

type keyringClient interface {
	Get(service, user string) (string, error)
	Set(service, user, secret string) error
	Delete(service, user string) error
}

type keyringAdapter struct{}

func (keyringAdapter) Get(service, user string) (string, error) {
	return keyring.Get(service, user)
}

func (keyringAdapter) Set(service, user, secret string) error {
	return keyring.Set(service, user, secret)
}

func (keyringAdapter) Delete(service, user string) error {
	return keyring.Delete(service, user)
}

type KeyringStore struct {
	service string
	user    string
	client  keyringClient
}

func NewKeyringStore(service, user string) *KeyringStore {
	return &KeyringStore{
		service: service,
		user:    user,
		client:  keyringAdapter{},
	}
}

func (s *KeyringStore) Load(context.Context) (config.StoredConfig, error) {
	secret, err := s.client.Get(s.service, s.user)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return config.StoredConfig{}, nil
		}
		return config.StoredConfig{}, err
	}
	var cfg config.StoredConfig
	if err := json.Unmarshal([]byte(secret), &cfg); err != nil {
		return config.StoredConfig{}, err
	}
	cfg.AuthStore = "keyring"
	return cfg, nil
}

func (s *KeyringStore) Save(_ context.Context, cfg config.StoredConfig) error {
	cfg.AuthStore = "keyring"
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return s.client.Set(s.service, s.user, string(data))
}

func (s *KeyringStore) Clear(context.Context) error {
	err := s.client.Delete(s.service, s.user)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}

func (s *KeyringStore) Kind() string {
	return "keyring"
}

type AutoStore struct {
	primary  TokenStore
	fallback TokenStore
	active   string
}

func NewAutoStore(primary, fallback TokenStore) *AutoStore {
	return &AutoStore{primary: primary, fallback: fallback, active: "auto"}
}

func (s *AutoStore) Load(ctx context.Context) (config.StoredConfig, error) {
	cfg, err := s.primary.Load(ctx)
	switch {
	case err == nil:
		s.active = kindOf(s.primary)
		return cfg, nil
	case errors.Is(err, keyring.ErrNotFound):
		cfg, fallbackErr := s.fallback.Load(ctx)
		if fallbackErr == nil {
			s.active = kindOf(s.fallback)
		}
		return cfg, fallbackErr
	case isKeyringUnavailable(err):
		cfg, fallbackErr := s.fallback.Load(ctx)
		if fallbackErr == nil {
			s.active = kindOf(s.fallback)
		}
		return cfg, fallbackErr
	default:
		return config.StoredConfig{}, err
	}
}

func (s *AutoStore) Save(ctx context.Context, cfg config.StoredConfig) error {
	cfg.AuthStore = kindOf(s.primary)
	if err := s.primary.Save(ctx, cfg); err == nil {
		s.active = kindOf(s.primary)
		_ = s.fallback.Clear(ctx)
		return nil
	} else if !isKeyringUnavailable(err) {
		return err
	}
	cfg.AuthStore = kindOf(s.fallback)
	if err := s.fallback.Save(ctx, cfg); err != nil {
		return err
	}
	s.active = kindOf(s.fallback)
	return nil
}

func (s *AutoStore) Clear(ctx context.Context) error {
	var errs []string
	if err := s.primary.Clear(ctx); err != nil {
		errs = append(errs, err.Error())
	}
	if err := s.fallback.Clear(ctx); err != nil {
		errs = append(errs, err.Error())
	}
	if len(errs) > 0 {
		return fmt.Errorf("clear auth stores: %s", strings.Join(errs, "; "))
	}
	s.active = "auto"
	return nil
}

func (s *AutoStore) Kind() string {
	if s.active != "" && s.active != "auto" {
		return s.active
	}
	return "auto"
}

func kindOf(store TokenStore) string {
	if named, ok := store.(StoreDescriptor); ok {
		return named.Kind()
	}
	return "unknown"
}

func isKeyringUnavailable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, keyring.ErrUnsupportedPlatform) || errors.Is(err, ErrKeyringNotImplemented) {
		return true
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "dbus") || strings.Contains(lower, "secret service") || strings.Contains(lower, "credential") || strings.Contains(lower, "keyring")
}

func KeyringServiceName() string {
	return "yandex-tracker-cli"
}

func KeyringUserName(configPath string) string {
	clean := filepath.Clean(configPath)
	clean = strings.ReplaceAll(clean, "\\", "_")
	clean = strings.ReplaceAll(clean, "/", "_")
	return "config:" + clean
}
