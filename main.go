package main

import (
	"email_test_app/backend/wails_app"
	"embed"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var icon []byte

func main() {
	// Create an instance of the app structure
	app := wails_app.NewApp()

	godotenv.Load()

	fmt.Println("GOOGLE_CLIENT_ID:", os.Getenv("GOOGLE_CLIENT_ID"))

	// Create application with options
	err := wails.Run(&options.App{
		Title:            "new_name",
		MinWidth:         400,
		MinHeight:        400,
		Assets:           assets,
		BackgroundColour: &options.RGBA{R: 0, G: 0, B: 0, A: 0},
		OnStartup:        app.Startup,
		DisableResize:    false,
		Windows: &windows.Options{
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
		},
		Mac: &mac.Options{
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			TitleBar:             mac.TitleBarHiddenInset(),
			About: &mac.AboutInfo{
				Title:   "email_test_app",
				Message: "2024 Kade Angell",
				Icon:    icon,
			},
		},
		Bind: []interface{}{
			app,
		},
		Logger:        &logger.DefaultLogger{},
		LogLevel:      logger.ERROR,
		OnDomReady:    app.DomReady,
		OnShutdown:    app.Shutdown,
		OnBeforeClose: app.BeforeClose,
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
