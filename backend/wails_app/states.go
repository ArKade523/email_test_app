package wails_app

import (
	"context"
)

// startup is called at application startup
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.oauthState = "state-token"
	a.oauthCodeChannel = make(chan string)

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
