package main

import (
	"context"
	"fmt"
	"os"

	"github.com/liangyou/govm/internal/cli"
	"github.com/liangyou/govm/internal/env"
	"github.com/liangyou/govm/internal/platform"
	"github.com/liangyou/govm/internal/region"
	"github.com/liangyou/govm/internal/remote"
	"github.com/liangyou/govm/internal/storage"
	"github.com/liangyou/govm/internal/version"
	"github.com/liangyou/govm/pkg/models"
)

const appVersion = "0.1.0"

func main() {
	cfg := models.Config{}
	checker := platform.NewChecker(cfg)
	if err := checker.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	store := storage.NewFileStorage(cfg)

	detector := region.NewDetector()
	countryCode, err := detector.CountryCode(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "warn: detect region failed, fallback to default source: %v\n", err)
	}
	mirror := region.SelectMirror(countryCode)

	remoteClient := remote.NewClient(
		remote.WithBaseURL(mirror.APIBase),
		remote.WithDownloadBase(mirror.DownloadBase),
	)
	downloader := version.NewDownloader(cfg)
	installer := version.NewInstaller(store, downloader)
	envManager := env.NewManager(store, cfg)
	switcher := version.NewSwitcher(store, envManager)
	uninstaller := version.NewUninstaller(store)
	lister := version.NewLister(remoteClient, store)

	app := cli.NewApp(os.Stdout, lister, installer, switcher, uninstaller, appVersion)
	if err := app.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
