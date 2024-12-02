package wails_app

import (
	"context"
	"database/sql"
	"email_test_app/backend/mail"
	"log"
	"net/http"
	"sync"
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
	ctx               context.Context
	imapUrl           string
	emailAddr         string
	emailAppPassword  string
	mailboxCache      []string
	mailboxCacheTime  time.Time
	mailboxCacheMutex sync.Mutex

	oauthToken       *oauth2.Token
	oauthState       string
	oauthCodeChannel chan string
	httpServer       *http.Server

	emailCache      map[string][]mail.SerializableMessage
	emailCacheTimes map[string]time.Time
	emailCacheMutex sync.Mutex

	emailBodyCache      map[string]map[uint32]EmailBodyCacheEntry // New cache for email bodies
	emailBodyCacheMutex sync.Mutex                                // Mutex for emailBodyCache
	db                  *sql.DB
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		emailCache:      make(map[string][]mail.SerializableMessage),
		emailCacheTimes: make(map[string]time.Time),
		emailBodyCache:  make(map[string]map[uint32]EmailBodyCacheEntry), // Initialize the email body cache
	}
}

func (a *App) LoginUserWithOAuth(providerName string) bool {
	err := a.StartOAuth(providerName)
	if err != nil {
		log.Println("OAuth login failed:", err)
		return false
	}
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
	return true
}

func (a *App) IsLoggedIn() bool {
	return a.emailAppPassword != "" || (a.oauthToken != nil && a.emailAddr != "")
}

func (a *App) LogoutUser() {
	a.imapUrl = ""
	a.emailAddr = ""
	a.emailAppPassword = ""

	// Clear caches
	a.mailboxCacheMutex.Lock()
	a.mailboxCache = nil
	a.mailboxCacheMutex.Unlock()

	a.emailCacheMutex.Lock()
	a.emailCache = make(map[string][]mail.SerializableMessage)
	a.emailCacheTimes = make(map[string]time.Time)
	a.emailCacheMutex.Unlock()

	a.emailBodyCacheMutex.Lock()
	a.emailBodyCache = make(map[string]map[uint32]EmailBodyCacheEntry) // Clear the email body cache
	a.emailBodyCacheMutex.Unlock()

	runtime.EventsEmit(a.ctx, "UserLoggedOut", nil)
}
