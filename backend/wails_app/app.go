package wails_app

import (
	"context"
	"database/sql"
	"email_test_app/backend/auth"
	"email_test_app/backend/mail"
	"fmt"
	"log"
	"net/http"
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
	ctx      context.Context
	accounts map[int64]auth.Account

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

func (a *App) LoginUser(imapUrl, emailAddr, emailAppPassword string) int64 {
	// Attempt to connect and login with the provided credentials
	err := mail.WithClient(imapUrl, emailAddr, emailAppPassword, func(c *client.Client) error {
		// Connection and login successful
		return nil
	})

	if err != nil {
		log.Println("Login failed:", err)
		return -1
	}

	// Store the credentials in the App struct
	newAccount := auth.Account{
		Email:               emailAddr,
		ImapUrl:             imapUrl,
		AppSpecificPassword: emailAppPassword,
	}

	err = a.updateAccounts(&newAccount)
	if err != nil {
		log.Println("Error updating accounts:", err)
		return -1
	}

	a.startUpdateLoops()

	return newAccount.Id
}

func (a *App) GetAccountIds() []int64 {
	ids := make([]int64, 0, len(a.accounts))
	for _, account := range a.accounts {
		ids = append(ids, account.Id)
	}
	return ids
}

func (a *App) IsLoggedIn(accountId int64) bool {
	_, ok := a.accounts[accountId]
	if !ok {
		log.Println("Account not found for ID:", accountId)
		log.Println("Accounts:", a.accounts)
	}

	return ok
}

func (a *App) updateAccounts(newAccount *auth.Account) error {
	// Update the accounts in the DB
	for _, account := range a.accounts {
		log.Println("Checking account:", account)
		if account.Email == newAccount.Email {
			var err error
			log.Println("Updating account:", account, "with new account:", newAccount)
			// update the account
			result, err := a.db.Exec(`
				UPDATE accounts 
				SET imap_url = ?, oauth_access_token = ?, oauth_refresh_token = ?, oauth_expiry = ?, app_specific_password = ?
				WHERE email = ?
			`, newAccount.ImapUrl, newAccount.OAuthAccessToken, newAccount.OAuthRefreshToken, newAccount.OAuthExpiry, newAccount.AppSpecificPassword, newAccount.Email)
			if err != nil {
				return fmt.Errorf("error updating accounts in the database: %v", err)
			}

			newAccount.Id, err = result.LastInsertId()
			if err != nil {
				return fmt.Errorf("error getting account ID: %v", err)
			}

			account = *newAccount

			// Update the accounts map
			a.accounts[account.Id] = account
			return nil
		}
	}

	// Insert the new account
	var err error
	result, err := a.db.Exec(`
		INSERT INTO accounts (email, imap_url, oauth_access_token, oauth_refresh_token, oauth_expiry, app_specific_password)
		VALUES (?, ?, ?, ?, ?, ?)
	`, newAccount.Email, newAccount.ImapUrl, newAccount.OAuthAccessToken, newAccount.OAuthRefreshToken, newAccount.OAuthExpiry, newAccount.AppSpecificPassword)
	if err != nil {
		return fmt.Errorf("error inserting account into the database: %v", err)
	}

	newAccount.Id, err = result.LastInsertId()
	if err != nil {
		return fmt.Errorf("error getting account ID: %v", err)
	}

	// Update the accounts map
	a.accounts[newAccount.Id] = *newAccount

	log.Println("Accounts Updated. Accounts:", a.accounts)

	return nil
}

func (a *App) LogoutUser(accountId int64) {
	// Remove the account's tokens and password from the App struct and database
	account, ok := a.accounts[accountId]
	if !ok {
		log.Println("Account not found.")
		return
	}

	account.OAuthAccessToken = ""
	account.OAuthRefreshToken = ""
	account.OAuthExpiry = 0
	account.AppSpecificPassword = ""

	a.updateAccounts(&account)

	a.endUpdateLoops(accountId)

	runtime.EventsEmit(a.ctx, "UserLoggedOut", accountId)
}
