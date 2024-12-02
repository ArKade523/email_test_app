package wails_app

import (
	"context"
	"email_test_app/backend/assets"
	"email_test_app/backend/auth"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/oauth2"
)

func (a *App) StartOAuth(providerName string) error {
	var oauthConfig *oauth2.Config
	var codeVerifier string
	var err error

	switch providerName {
	case "Gmail":
		oauthConfig = auth.GmailOAuthConfig
	default:
		return fmt.Errorf("unsupported provider for OAuth")
	}

	codeVerifier, err = auth.GenerateCodeVerifier()
	if err != nil {
		return err
	}
	codeChallenge := auth.GenerateCodeChallenge(codeVerifier)

	authURL := oauthConfig.AuthCodeURL(a.oauthState, oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"))

	// Open the browser to let the user authenticate
	runtime.BrowserOpenURL(a.ctx, authURL)

	// Wait for the authorization code from the HTTP handler
	code := <-a.oauthCodeChannel

	// Exchange the authorization code for a token
	token, err := oauthConfig.Exchange(context.Background(), code,
		oauth2.SetAuthURLParam("code_verifier", codeVerifier))
	if err != nil {
		log.Println("Token exchange failed:", err)
		runtime.EventsEmit(a.ctx, "OAuthFailure", nil)
		return err
	}

	// Store the token
	a.oauthToken = token

	// Get the user's email address
	client := oauthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		log.Println("Failed to get user info:", err)
		runtime.EventsEmit(a.ctx, "OAuthFailure", nil)
		return err
	}
	defer resp.Body.Close()

	var userInfo struct {
		Email string `json:"email"`
	}
	err = json.NewDecoder(resp.Body).Decode(&userInfo)
	if err != nil {
		log.Println("Failed to parse user info:", err)
		runtime.EventsEmit(a.ctx, "OAuthFailure", nil)
		return err
	}

	a.emailAddr = userInfo.Email
	a.imapUrl = "imap.gmail.com:993"

	// Emit an event to the frontend to proceed
	runtime.EventsEmit(a.ctx, "OAuthSuccess", nil)

	return nil
}

func (a *App) oauthCallbackHandler(w http.ResponseWriter, r *http.Request) {
	// Validate state parameter
	state := r.URL.Query().Get("state")
	if state != a.oauthState {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "No code in request", http.StatusBadRequest)
		runtime.EventsEmit(a.ctx, "OAuthFailure", nil)
		return
	}

	// Send the code back to the application
	a.oauthCodeChannel <- code

	htmlContent, err := assets.OauthSuccessHTML.ReadFile("oauth_success.html")
	if err != nil {
		http.Error(w, "Unable to load HTML", http.StatusInternalServerError)
		return
	}

	// Set the Content-Type to text/html
	w.Header().Set("Content-Type", "text/html")
	// Write the file content as the response
	fmt.Fprint(w, string(htmlContent))
}

func (a *App) startHTTPServer() {
	http.HandleFunc("/oauth2callback", a.oauthCallbackHandler)
	http.HandleFunc("/appicon.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(assets.AppIconPNG)
	})

	a.httpServer = &http.Server{Addr: "localhost:9498"}

	// Start the server
	if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("HTTP server failed: %v", err)
	}
}
