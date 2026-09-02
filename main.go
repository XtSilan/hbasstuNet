package main

import (
	"context"
	"embed"
	"log"

	"hbasstuNet/internal/applog"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
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

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "hbasstuNet",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
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
