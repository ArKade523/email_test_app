package wails_app

import (
	"email_test_app/backend/auth"
	"email_test_app/backend/mail"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/emersion/go-imap/client"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) startUpdateLoops() {
	a.mailboxUpdateTicker.Reset(MAILBOX_UPDATE_TIME)
	a.emailUpdateTicker.Reset(EMAIL_UPDATE_TIME)

	// Run once, immediately
	go func() {
		a.UpdateMailboxes()
		for _, mailbox := range a.GetMailboxes() {
			a.UpdateMessages(mailbox)
		}
	}()

	go func() {
		for range a.mailboxUpdateTicker.C {
			a.UpdateMailboxes()
		}
	}()

	go func() {
		for range a.emailUpdateTicker.C {
			mailboxes := a.GetMailboxes()
			for _, mailbox := range mailboxes {
				a.UpdateMessages(mailbox)
			}
		}
	}()
}

func (a *App) endUpdateLoops() {
	a.mailboxUpdateTicker.Stop()
	a.emailUpdateTicker.Stop()
}

var mailboxUpdateMutex sync.Mutex

func (a *App) UpdateMailboxes() {
	if !a.IsLoggedIn() {
		log.Println("UpdateMailboxes: User not logged in.")
		return
	}

	// use a mutex to prevent multiple updates at the same time
	if !mailboxUpdateMutex.TryLock() {
		log.Println("UpdateMailboxes: Update already in progress.")
		return
	}
	defer mailboxUpdateMutex.Unlock()

	var mailboxes []string
	var err error

	if a.oauthToken != nil {
		oauthConfig := auth.GmailOAuthConfig
		_, err = mail.WithOAuthClient(a.imapUrl, a.emailAddr, a.oauthToken, oauthConfig, func(c *client.Client) error {
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
	} else {
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
		log.Println("Error fetching mailboxes from server:", err)
		return
	}

	tx, err := a.db.Begin()
	if err != nil {
		log.Println("Error starting transaction to update mailboxes:", err)
		return
	}
	defer tx.Rollback()

	// check if the existing mailboxes are the same as the new ones
	mailboxesMatch := true
	existingMailboxes := a.GetMailboxes()
	if len(existingMailboxes) == len(mailboxes) {
		for _, name := range mailboxes {
			nameFound := false
			for _, existingName := range existingMailboxes {
				if name == existingName {
					nameFound = true
					break
				}
			}
			if !nameFound {
				mailboxesMatch = false
				break
			}
		}
	} else {
		mailboxesMatch = false
	}

	if !mailboxesMatch {
		// Clear the existing mailboxes
		_, err = tx.Exec("DELETE FROM mailboxes")
		if err != nil {
			log.Println("Error clearing mailboxes:", err)
			return
		}

		stmt, err := tx.Prepare("INSERT INTO mailboxes (name) VALUES (?)")
		if err != nil {
			log.Println("Error preparing statement to insert mailboxes:", err)
			return
		}
		defer stmt.Close()

		for _, name := range mailboxes {
			_, err = stmt.Exec(name)
			if err != nil {
				log.Println("Error inserting mailbox:", err)
			}
		}

		if err := tx.Commit(); err != nil {
			log.Println("Error committing transaction to update mailboxes:", err)
			return
		}

		mailboxesFromDB := a.GetMailboxes()
		log.Println("Mailboxes updated:", mailboxesFromDB)

		runtime.EventsEmit(a.ctx, "MailboxesUpdated")
	} else {
		log.Println("Mailboxes match, not updating.")
	}
}

var messageUpdateMutex sync.Mutex

func (a *App) UpdateMessages(mailboxName string) {
	if !a.IsLoggedIn() {
		log.Println("UpdateMessages: User not logged in.")
		return
	}

	// use a mutex to prevent multiple updates at the same time
	if !messageUpdateMutex.TryLock() {
		log.Println("UpdateMessages: Update already in progress.")
		return
	}
	defer messageUpdateMutex.Unlock()

	var messages []mail.SerializableMessage
	var err error

	if a.oauthToken != nil {
		oauthConfig := auth.GmailOAuthConfig
		_, err = mail.WithOAuthClient(a.imapUrl, a.emailAddr, a.oauthToken, oauthConfig, func(c *client.Client) error {
			emails, err := mail.FetchEmailsForMailbox(c, mailboxName, 0, 100) // Adjust range as needed
			if err != nil {
				return err
			}
			messages = emails
			return nil
		})
	} else {
		err = mail.WithClient(a.imapUrl, a.emailAddr, a.emailAppPassword, func(c *client.Client) error {
			emails, err := mail.FetchEmailsForMailbox(c, mailboxName, 0, 100) // Adjust range as needed
			if err != nil {
				return err
			}
			for i, email := range emails {
				email.Body, err = mail.FetchEmailBody(c, email.UID)
				if err != nil {
					log.Println("Error fetching email body:", err)
					continue
				}
				emails[i] = email
			}

			messages = emails
			return nil
		})
	}

	if err != nil {
		log.Println("Error fetching messages from server:", err)
		return
	}

	tx, err := a.db.Begin()
	if err != nil {
		log.Println("Error starting transaction to update messages:", err)
		return
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
        INSERT OR REPLACE INTO messages (mailbox_name, uid, envelope, body, received_at, last_updated) 
        VALUES (?, ?, ?, ?, ?, ?)
    `)
	if err != nil {
		log.Println("Error preparing statement to insert messages:", err)
		return
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
		log.Println("Error committing transaction to update messages:", err)
		return
	}

	runtime.EventsEmit(a.ctx, "MessagesUpdated", mailboxName)
}
