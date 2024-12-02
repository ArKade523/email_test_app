package wails_app

import (
	"context"
	"email_test_app/backend/db"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

// startup is called at application startup
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.oauthState = "state-token"
	a.oauthCodeChannel = make(chan string)

	appDataDir, err := getAppDataDir()
	if err != nil {
		log.Println("Error getting app data directory:", err)
		return
	}

	a.db, err = db.InitDB(appDataDir + "/email_test_app.db")
	if err != nil {
		log.Println("Error initializing database:", err)
		return
	}

	go a.startHTTPServer()
}

// domReady is called after front-end resources have been loaded
func (a *App) DomReady(ctx context.Context) {
	// Add your action here
}

// beforeClose is called when the application is about to quit
func (a *App) BeforeClose(ctx context.Context) (prevent bool) {
	return false
}

// shutdown is called at application termination
func (a *App) Shutdown(ctx context.Context) {
	// Perform your teardown here
	a.LogoutUser()

	// Shutdown the HTTP server
	if a.httpServer != nil {
		a.httpServer.Shutdown(ctx)
	}
}

func getAppDataDir() (string, error) {
	var baseDir string
	appName := "EmailTestApp"

	switch runtime.GOOS {
	case "darwin": // macOS
		baseDir = filepath.Join(os.Getenv("HOME"), "Library", "Application Support")
	case "linux": // Linux
		baseDir = filepath.Join(os.Getenv("HOME"), ".local", "share")
	case "windows": // Windows
		baseDir = os.Getenv("APPDATA") // Typically resolves to %APPDATA%
	default:
		return "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	appDir := filepath.Join(baseDir, appName)

	// Ensure the directory exists
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create app data directory: %w", err)
	}

	return appDir, nil
}
