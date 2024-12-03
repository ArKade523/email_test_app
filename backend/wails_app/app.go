package wails_app

import (
	"context"
	"database/sql"
	"email_test_app/backend/mail"
	"log"
	"net/http"
	"time"

	"github.com/emersion/go-imap/client"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/oauth2"
)

type EmailBodyCacheEntry struct {
	Body      string
	CacheTime time.Time
}

// App struct
type App struct {
	ctx              context.Context
	imapUrl          string
	emailAddr        string
	emailAppPassword string

	oauthToken       *oauth2.Token
	oauthState       string
	oauthCodeChannel chan string
	httpServer       *http.Server

	mailboxUpdateTicker *time.Ticker
	emailUpdateTicker   *time.Ticker

	db *sql.DB
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

func (a *App) LoginUserWithOAuth(providerName string) bool {
	err := a.StartOAuth(providerName)
	if err != nil {
		log.Println("OAuth login failed:", err)
		return false
	}

	a.startUpdateLoops()

	return true
}

func (a *App) LoginUser(imapUrl, emailAddr, emailAppPassword string) bool {
	// Attempt to connect and login with the provided credentials
	err := mail.WithClient(imapUrl, emailAddr, emailAppPassword, func(c *client.Client) error {
		// Connection and login successful
		return nil
	})

	if err != nil {
		log.Println("Login failed:", err)
		return false
	}

	// Store the credentials in the App struct
	a.imapUrl = imapUrl
	a.emailAddr = emailAddr
	a.emailAppPassword = emailAppPassword

	a.startUpdateLoops()

	return true
}

func (a *App) IsLoggedIn() bool {
	return a.emailAppPassword != "" || (a.oauthToken != nil && a.emailAddr != "")
}

func (a *App) LogoutUser() {
	a.imapUrl = ""
	a.emailAddr = ""
	a.emailAppPassword = ""

	a.endUpdateLoops()

	runtime.EventsEmit(a.ctx, "UserLoggedOut", nil)
}
