package wails_app

import (
	"database/sql"
	"email_test_app/backend/auth"
	"email_test_app/backend/mail"
	"encoding/json"
	"log"
	"time"

	"github.com/emersion/go-imap/client"
	"golang.org/x/oauth2"
)

const MAILBOX_CACHE_TIME = 5 * time.Minute
const EMAIL_CACHE_TIME = 5 * time.Minute

func (a *App) GetMailboxes() []string {
	if !a.IsLoggedIn() {
		log.Println("GetMailboxes: User not logged in.")
		a.LogoutUser()
		return nil
	}

	const SQLiteTimeFormat = "2006-01-02 15:04:05.999999-07:00"

	var mailboxes []string
	var lastUpdated sql.NullString // Use sql.NullString to handle NULL values

	row := a.db.QueryRow("SELECT MAX(last_updated) FROM mailboxes")
	if err := row.Scan(&lastUpdated); err != nil && err != sql.ErrNoRows {
		log.Println("Error checking mailbox cache timestamp:", err)
	}

	var cacheValid bool
	if lastUpdated.Valid { // Check if last_updated is not NULL
		parsedTime, err := time.Parse(SQLiteTimeFormat, lastUpdated.String)
		if err != nil {
			log.Println("Error parsing last_updated time:", err)
		} else if time.Since(parsedTime) < EMAIL_CACHE_TIME {
			cacheValid = true
		}
	}

	if cacheValid {
		rows, err := a.db.Query("SELECT name FROM mailboxes")
		if err != nil {
			log.Println("Error querying mailboxes from database:", err)
			return nil
		}
		defer rows.Close()

		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				log.Println("Error scanning mailbox row:", err)
				continue
			}
			mailboxes = append(mailboxes, name)
		}

		if len(mailboxes) > 0 {
			log.Println("Returning mailboxes from SQLite cache.")
			return mailboxes
		}
	}

	// Fetch mailboxes from the IMAP server
	var errFetch error
	if a.oauthToken != nil {
		// Use OAuth client
		oauthConfig := auth.GmailOAuthConfig
		var newToken *oauth2.Token
		newToken, errFetch = mail.WithOAuthClient(a.imapUrl, a.emailAddr, a.oauthToken, oauthConfig, func(c *client.Client) error {
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
		errFetch = mail.WithClient(a.imapUrl, a.emailAddr, a.emailAppPassword, func(c *client.Client) error {
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

	if errFetch != nil {
		log.Println("Error fetching mailboxes from server:", errFetch)
		return nil
	}

	// Update the SQLite database with the fetched mailboxes
	tx, err := a.db.Begin()
	if err != nil {
		log.Println("Error starting transaction to store mailboxes:", err)
		return mailboxes
	}
	defer tx.Rollback()

	// Clear existing mailboxes
	_, err = tx.Exec("DELETE FROM mailboxes")
	if err != nil {
		log.Println("Error clearing mailboxes:", err)
		return mailboxes
	}

	// Insert new mailboxes
	stmt, err := tx.Prepare("INSERT INTO mailboxes (account_id, name, last_updated) VALUES (0, ?, ?)") // TODO: account_id
	if err != nil {
		log.Println("Error preparing statement to insert mailboxes:", err)
		return mailboxes
	}
	defer stmt.Close()

	for _, name := range mailboxes {
		_, err = stmt.Exec(name, time.Now())
		if err != nil {
			log.Println("Error inserting mailbox into cache:", err)
		}
	}

	if err := tx.Commit(); err != nil {
		log.Println("Error committing transaction to store mailboxes:", err)
	}

	return mailboxes
}

// GetEmailsForMailbox returns emails for a mailbox, using cache if available
func (a *App) GetEmailsForMailbox(mailboxName string, start, limit uint32) []mail.SerializableMessage {
	if !a.IsLoggedIn() {
		log.Println("GetEmailsForMailbox: User not logged in.")
		a.LogoutUser()
		return nil
	}

	var messages []mail.SerializableMessage
	var lastUpdated sql.NullString // Use sql.NullString to handle NULL values

	row := a.db.QueryRow("SELECT MAX(last_updated) FROM messages WHERE mailbox_name = ?", mailboxName)
	if err := row.Scan(&lastUpdated); err != nil {
		log.Println("Error checking message cache timestamp:", err)
	}

	var cacheValid bool
	if lastUpdated.Valid { // Check if last_updated is not NULL
		parsedTime, err := time.Parse(time.RFC3339, lastUpdated.String)
		if err != nil {
			log.Println("Error parsing last_updated time:", err)
		} else if time.Since(parsedTime) < EMAIL_CACHE_TIME {
			cacheValid = true
		}
	}

	if cacheValid {
		rows, err := a.db.Query(`
            SELECT uid, envelope FROM messages 
            WHERE mailbox_name = ? 
            ORDER BY received_at DESC 
            LIMIT ? OFFSET ?
        `, mailboxName, limit, start)
		if err != nil {
			log.Println("Error querying messages from database:", err)
			return nil
		}
		defer rows.Close()

		for rows.Next() {
			var msg mail.SerializableMessage
			var envelopeData []byte
			if err := rows.Scan(&msg.UID, &envelopeData); err != nil {
				log.Println("Error scanning message row:", err)
				continue
			}

			if err := json.Unmarshal(envelopeData, &msg.Envelope); err != nil {
				log.Println("Error unmarshalling envelope:", err)
				continue
			}

			msg.MailboxName = mailboxName
			msg.Body = "" // Ensure the body is not sent to the frontend
			messages = append(messages, msg)
		}

		if len(messages) > 0 {
			log.Println("Returning messages from SQLite cache.")
			return messages
		}
	}

	// Fetch emails from the IMAP server
	var errFetch error
	if a.oauthToken != nil {
		// Use OAuth client
		oauthConfig := auth.GmailOAuthConfig
		var newToken *oauth2.Token
		newToken, errFetch = mail.WithOAuthClient(a.imapUrl, a.emailAddr, a.oauthToken, oauthConfig, func(c *client.Client) error {
			emails, err := mail.FetchEmailsForMailbox(c, mailboxName, start, limit)
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
		errFetch = mail.WithClient(a.imapUrl, a.emailAddr, a.emailAppPassword, func(c *client.Client) error {
			emails, err := mail.FetchEmailsForMailbox(c, mailboxName, start, limit)
			if err != nil {
				return err
			}
			messages = emails
			return nil
		})
	}

	if errFetch != nil {
		log.Println("Error fetching messages from server:", errFetch)
		return nil
	}

	// Store messages in the database
	tx, err := a.db.Begin()
	if err != nil {
		log.Println("Error starting transaction to store messages:", err)
		return messages
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
        INSERT OR REPLACE INTO messages (mailbox_name, uid, envelope, body, received_at, last_updated) 
        VALUES (?, ?, ?, ?, ?, ?)
    `)
	if err != nil {
		log.Println("Error preparing statement to insert messages:", err)
		return messages
	}
	defer stmt.Close()

	for _, msg := range messages {
		envelopeData, err := json.Marshal(msg.Envelope)
		if err != nil {
			log.Println("Error marshalling envelope:", err)
			continue
		}

		_, err = stmt.Exec(mailboxName, msg.UID, envelopeData, msg.Body, time.Now(), time.Now())
		if err != nil {
			log.Println("Error inserting message into database:", err)
		}
	}

	if err := tx.Commit(); err != nil {
		log.Println("Error committing transaction to store messages:", err)
	}

	// Remove the body from messages before returning to the frontend
	for i := range messages {
		messages[i].Body = ""
	}

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
