package wails_app

import (
	"email_test_app/backend/auth"
	"email_test_app/backend/mail"
	"log"
	"time"

	"github.com/emersion/go-imap/client"
	"golang.org/x/oauth2"
)

func (a *App) GetMailboxes() []string {
	if !a.IsLoggedIn() {
		log.Println("GetMailboxes: User not logged in.")
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
	var err error

	if a.oauthToken != nil {
		// Use OAuth client
		oauthConfig := auth.GmailOAuthConfig
		var newToken *oauth2.Token
		newToken, err = mail.WithOAuthClient(a.imapUrl, a.emailAddr, a.oauthToken, oauthConfig, func(c *client.Client) error {
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
		if newToken != nil && newToken.AccessToken != a.oauthToken.AccessToken {
			a.oauthToken = newToken
		}
	} else {
		// Use regular client
		err = mail.WithClient(a.imapUrl, a.emailAddr, a.emailAppPassword, func(c *client.Client) error {
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
	}

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
	if !a.IsLoggedIn() {
		log.Println("GetEmailsForMailbox: User not logged in.")
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
	var err error

	if a.oauthToken != nil {
		// Use OAuth client
		oauthConfig := auth.GmailOAuthConfig
		var newToken *oauth2.Token
		newToken, err = mail.WithOAuthClient(a.imapUrl, a.emailAddr, a.oauthToken, oauthConfig, func(c *client.Client) error {
			emails, err := mail.FetchEmailsForMailbox(c, mailboxName)
			if err != nil {
				return err
			}
			messages = emails
			return nil
		})
		if newToken != nil && newToken.AccessToken != a.oauthToken.AccessToken {
			a.oauthToken = newToken
		}
	} else {
		// Use regular client
		err = mail.WithClient(a.imapUrl, a.emailAddr, a.emailAppPassword, func(c *client.Client) error {
			emails, err := mail.FetchEmailsForMailbox(c, mailboxName)
			if err != nil {
				return err
			}
			messages = emails
			return nil
		})
	}

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
	if !a.IsLoggedIn() {
		log.Println("GetEmailBody: User not logged in.")
		a.LogoutUser()
		return ""
	}

	var body string

	// Check the cache
	a.emailBodyCacheMutex.Lock()
	if a.emailBodyCache[mailboxName] == nil {
		a.emailBodyCache[mailboxName] = make(map[uint32]EmailBodyCacheEntry)
	}
	cacheEntry, exists := a.emailBodyCache[mailboxName][seqNum]
	a.emailBodyCacheMutex.Unlock()

	if exists && time.Since(cacheEntry.CacheTime) < EMAIL_CACHE_TIME {
		// Cache hit and cache is valid
		return cacheEntry.Body
	}

	var err error

	if a.oauthToken != nil {
		// Use OAuth client
		oauthConfig := auth.GmailOAuthConfig
		var newToken *oauth2.Token
		newToken, err = mail.WithOAuthClient(a.imapUrl, a.emailAddr, a.oauthToken, oauthConfig, func(c *client.Client) error {
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
		if newToken != nil && newToken.AccessToken != a.oauthToken.AccessToken {
			a.oauthToken = newToken
		}
	} else {
		// Use regular client
		err = mail.WithClient(a.imapUrl, a.emailAddr, a.emailAppPassword, func(c *client.Client) error {
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
	}

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
