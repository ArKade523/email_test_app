package main

import (
	"context"
	"email_test_app/mail"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/emersion/go-imap/client"
	"github.com/wailsapp/wails/v2/pkg/runtime"
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

	emailCache      map[string][]mail.SerializableMessage
	emailCacheTimes map[string]time.Time
	emailCacheMutex sync.Mutex

	emailBodyCache      map[string]map[uint32]EmailBodyCacheEntry // New cache for email bodies
	emailBodyCacheMutex sync.Mutex                                // Mutex for emailBodyCache
}

const MAILBOX_CACHE_TIME = 5 * time.Minute
const EMAIL_CACHE_TIME = 5 * time.Minute

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		emailCache:      make(map[string][]mail.SerializableMessage),
		emailCacheTimes: make(map[string]time.Time),
		emailBodyCache:  make(map[string]map[uint32]EmailBodyCacheEntry), // Initialize the email body cache
	}
}

// startup is called at application startup
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// No longer need to load .env or initialize credentials here
}

// GetMailboxes returns a list of mailboxes, using cache if available
func (a *App) GetMailboxes() []string {
	if a.emailAppPassword == "" {
		log.Println("User not logged in.")
		a.LogoutUser()
		return nil
	}

	a.mailboxCacheMutex.Lock()
	defer a.mailboxCacheMutex.Unlock()

	// Check if cache is valid
	if time.Since(a.mailboxCacheTime) < MAILBOX_CACHE_TIME && a.mailboxCache != nil {
		return a.mailboxCache
	}

	var mailboxes []string
	err := mail.WithClient(a.imapUrl, a.emailAddr, a.emailAppPassword, func(c *client.Client) error {
		mboxes, err := mail.FetchMailboxes(c)
		if err != nil {
			return err
		}

		mailboxes = make([]string, len(mboxes))
		for i, mbox := range mboxes {
			mailboxes[i] = mbox.Name
		}

		return nil
	})

	if err != nil {
		log.Println("Error fetching mailboxes:", err)
		return nil
	}

	// Update cache
	a.mailboxCache = mailboxes
	a.mailboxCacheTime = time.Now()

	return mailboxes
}

// GetEmailsForMailbox returns emails for a mailbox, using cache if available
func (a *App) GetEmailsForMailbox(mailboxName string) []mail.SerializableMessage {
	if a.emailAppPassword == "" {
		log.Println("User not logged in.")
		a.LogoutUser()
		return nil
	}

	a.emailCacheMutex.Lock()
	defer a.emailCacheMutex.Unlock()

	// Check if cache is valid
	if cacheTime, exists := a.emailCacheTimes[mailboxName]; exists {
		if time.Since(cacheTime) < EMAIL_CACHE_TIME {
			if cachedEmails, ok := a.emailCache[mailboxName]; ok {
				return cachedEmails
			}
		}
	}

	var messages []mail.SerializableMessage
	err := mail.WithClient(a.imapUrl, a.emailAddr, a.emailAppPassword, func(c *client.Client) error {
		emails, err := mail.FetchEmailsForMailbox(c, mailboxName)
		if err != nil {
			return err
		}
		messages = emails
		return nil
	})

	if err != nil {
		log.Println("Error fetching messages:", err)
		return nil
	}

	// Update cache
	a.emailCache[mailboxName] = messages
	a.emailCacheTimes[mailboxName] = time.Now()

	return messages
}

// GetEmailBody fetches the body of an email, using cache if available
func (a *App) GetEmailBody(mailboxName string, seqNum uint32) string {
	if a.emailAppPassword == "" {
		log.Println("User not logged in.")
		a.LogoutUser()
		return ""
	}

	var body string

	// Check the cache
	a.emailBodyCacheMutex.Lock()

	// Initialize the inner map if it doesn't exist
	if a.emailBodyCache[mailboxName] == nil {
		a.emailBodyCache[mailboxName] = make(map[uint32]EmailBodyCacheEntry)
	}

	cacheEntry, exists := a.emailBodyCache[mailboxName][seqNum]
	if exists && time.Since(cacheEntry.CacheTime) < EMAIL_CACHE_TIME {
		// Cache hit and cache is valid
		body = cacheEntry.Body
		a.emailBodyCacheMutex.Unlock()
		return body
	}
	a.emailBodyCacheMutex.Unlock()

	// Cache miss or expired, fetch from server
	err := mail.WithClient(a.imapUrl, a.emailAddr, a.emailAppPassword, func(c *client.Client) error {
		// Select the mailbox
		_, err := c.Select(mailboxName, false)
		if err != nil {
			return err
		}
		emailBody, err := mail.FetchEmailBody(c, seqNum)
		if err != nil {
			return err
		}
		body = emailBody
		return nil
	})

	if err != nil {
		log.Println("Error fetching email body:", err)
		return ""
	}

	// Cache the fetched email body
	a.emailBodyCacheMutex.Lock()
	a.emailBodyCache[mailboxName][seqNum] = EmailBodyCacheEntry{
		Body:      body,
		CacheTime: time.Now(),
	}
	a.emailBodyCacheMutex.Unlock()

	return body
}

// LoginUser handles user login
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

// LogoutUser handles user logout
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

// domReady is called after front-end resources have been loaded
func (a *App) domReady(ctx context.Context) {
	// Add your action here
}

// beforeClose is called when the application is about to quit
func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	return false
}

// shutdown is called at application termination
func (a *App) shutdown(ctx context.Context) {
	// Perform your teardown here
	a.LogoutUser()
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}
