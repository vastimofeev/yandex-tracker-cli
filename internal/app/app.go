package app

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/vasti/yandex-tracker-cli/internal/auth"
	"github.com/vasti/yandex-tracker-cli/internal/client"
	"github.com/vasti/yandex-tracker-cli/internal/config"
	"github.com/vasti/yandex-tracker-cli/internal/output"
)

type App struct {
	in         io.Reader
	out        io.Writer
	err        io.Writer
	httpClient *http.Client
}

type Runtime struct {
	Config  config.RuntimeConfig
	Store   auth.TokenStore
	Client  *client.Client
	Logger  *slog.Logger
	Printer output.Printer
	Out     io.Writer
	Err     io.Writer
	Input   io.Reader
}

func New(in io.Reader, out, err io.Writer) *App {
	return &App{
		in:         in,
		out:        out,
		err:        err,
		httpClient: &http.Client{},
	}
}

func (a *App) BuildRuntime(ctx context.Context, opts config.CLIOptions) (*Runtime, error) {
	configPath := opts.ConfigPath
	if configPath == "" {
		var err error
		configPath, err = config.DefaultPath()
		if err != nil {
			return nil, err
		}
	}
	store := buildStore(configPath, opts.AuthStore)
	stored, err := store.Load(ctx)
	if err != nil {
		return nil, err
	}
	opts.ConfigPath = configPath
	cfg, err := config.Resolve(opts, config.OSEnv, stored)
	if err != nil {
		return nil, err
	}
	if a.httpClient.Timeout == 0 {
		a.httpClient.Timeout = 30 * time.Second
	}

	level := slog.LevelWarn
	if cfg.Debug {
		level = slog.LevelDebug
	}
	handler := slog.NewTextHandler(a.err, &slog.HandlerOptions{Level: level})

	return &Runtime{
		Config:  cfg,
		Store:   buildStore(cfg.ConfigPath, cfg.AuthStore),
		Client:  client.New(cfg.BaseURL, cfg.Auth, a.httpClient),
		Logger:  slog.New(handler),
		Printer: output.NewPrinter(a.out, cfg.JSON),
		Out:     a.out,
		Err:     a.err,
		Input:   a.in,
	}, nil
}

func buildStore(configPath, requested string) auth.TokenStore {
	_ = requested
	return auth.NewKeyringStore(auth.KeyringServiceName(), auth.KeyringUserName(configPath))
}
