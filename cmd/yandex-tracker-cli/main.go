package main

import (
	"context"
	"fmt"
	"os"

	"github.com/vasti/yandex-tracker-cli/internal/app"
	"github.com/vasti/yandex-tracker-cli/internal/cli"
	"github.com/vasti/yandex-tracker-cli/internal/version"
)

var (
	buildVersion = "dev"
	buildCommit  = "unknown"
	buildDate    = "unknown"
)

func main() {
	version.Version = buildVersion
	version.Commit = buildCommit
	version.BuildDate = buildDate
	application := app.New(os.Stdin, os.Stdout, os.Stderr)
	rootCmd := cli.NewRootCommand(application)
	if err := rootCmd.ExecuteContext(context.Background()); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
