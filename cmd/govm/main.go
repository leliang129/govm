package main

import (
	"fmt"
	"os"

	"github.com/liangyou/govm/internal/cli"
	"github.com/liangyou/govm/internal/env"
	"github.com/liangyou/govm/internal/remote"
	"github.com/liangyou/govm/internal/storage"
	"github.com/liangyou/govm/internal/version"
	"github.com/liangyou/govm/pkg/models"
)

const appVersion = "0.1.0"

func main() {
	cfg := models.Config{}
	store := storage.NewFileStorage(cfg)
	remoteClient := remote.NewClient()
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
