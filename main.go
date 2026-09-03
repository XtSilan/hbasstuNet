package main

import (
	"context"
	"embed"
	"log"
	"os"
	"path/filepath"
	"slices"

	"hbasstuNet/internal/applog"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	logFile, logErr := applog.Init()
	if logErr == nil {
		defer logFile.Close()
		log.Printf("hbasstuNet starting; log=%s", applog.Path())
	}
	// Create an instance of the app structure
	app := NewApp()
	background := slices.Contains(os.Args[1:], "--background")
	webviewDataPath := filepath.Join(os.Getenv("APPDATA"), "hbasstuNet", "webview2")
	if err := os.MkdirAll(webviewDataPath, 0o755); err != nil {
		log.Printf("create WebView2 data directory failed: %v", err)
		webviewDataPath = ""
	}

	// Create application with options
	err := wails.Run(&options.App{
		Title:             "hbasstuNet",
		Width:             1024,
		Height:            768,
		MinWidth:          760,
		MinHeight:         560,
		StartHidden:       background,
		HideWindowOnClose: false,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 17, G: 18, B: 19, A: 1},
		Windows: &windows.Options{
			WebviewUserDataPath:  webviewDataPath,
			WebviewGpuIsDisabled: true,
			Theme:                windows.Dark,
		},
		OnStartup:     app.startup,
		OnShutdown:    func(context.Context) { app.stopTray() },
		OnBeforeClose: app.beforeClose,
		OnDomReady: func(context.Context) {
			log.Printf("frontend DOM ready")
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		log.Printf("application stopped with error: %v", err)
		println("Error:", err.Error())
	}
}
