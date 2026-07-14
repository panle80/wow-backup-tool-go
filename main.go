package main

import (
	"embed"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "WoW 插件备份恢复工具 v2.0.1",
		Width:     600,
		Height:    640,
		MinWidth:  520,
		MinHeight: 500,

		AssetServer: &assetserver.Options{
			Assets: assets,
		},

		// Background colour before the frontend loads
		BackgroundColour: &options.RGBA{R: 11, G: 13, B: 15, A: 1},

		OnStartup:  app.startup,
		OnShutdown: app.shutdown,

		// Bind all exported App methods so the frontend can call them
		Bind: []interface{}{
			app,
		},

		// Windows-specific
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
		},

		// Disable the default right-click context menu in the webview
		Debug: options.Debug{
			OpenInspectorOnStartup: false,
		},
	})

	if err != nil {
		println("Error:", err.Error())
		os.Exit(1)
	}
}
