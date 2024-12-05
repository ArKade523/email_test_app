package wails_app

import (
	"database/sql"
	"email_test_app/backend/auth"
	"email_test_app/backend/mail"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/emersion/go-imap"
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

	// go func() {
	// 	for range a.mailboxUpdateTicker.C {
	// 		a.UpdateMailboxes()
	// 	}
	// }()

	// go func() {
	// 	for range a.emailUpdateTicker.C {
	// 		mailboxes := a.GetMailboxes()
	// 		for _, mailbox := range mailboxes {
	// 			a.UpdateMessages(mailbox)
	// 		}
	// 	}
	// }()
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

	log.Println("Updating mailboxes")

	// use a mutex to prevent multiple updates at the same time
	if !mailboxUpdateMutex.TryLock() {
		log.Println("UpdateMailboxes: Update already in progress.")
		return
	}
	defer mailboxUpdateMutex.Unlock()

	var mailboxes []string
	var err error

	fetchMailboxes := func(c *client.Client) error {
		mboxes, err := mail.FetchMailboxes(c)
		if err != nil {
			return err
		}
		mailboxes = make([]string, len(mboxes))
		for i, mbox := range mboxes {
			mailboxes[i] = mbox.Name
		}
		return nil
	}

	if a.oauthToken != nil {
		oauthConfig := auth.GmailOAuthConfig
		_, err = mail.WithOAuthClient(a.imapUrl, a.emailAddr, a.oauthToken, oauthConfig, fetchMailboxes)
	} else {
		err = mail.WithClient(a.imapUrl, a.emailAddr, a.emailAppPassword, fetchMailboxes)
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

	log.Println("Updating messages for mailbox:", mailboxName)

	// Use a mutex to prevent multiple updates at the same time
	if !messageUpdateMutex.TryLock() {
		log.Println("UpdateMessages: Update already in progress.")
		return
	}
	defer messageUpdateMutex.Unlock()

	existingUIDs, err := fetchExistingUIDs(a.db, mailboxName)
	if err != nil {
		log.Println("Error fetching existing UIDs from database:", err)
		return
	}
	existingUIDSet := make(map[uint32]struct{}, len(existingUIDs))
	for _, uid := range existingUIDs {
		existingUIDSet[uid] = struct{}{}
	}

	var newMessages []mail.SerializableMessage

	fetchMessages := func(c *client.Client) error {
		mbox, err := c.Select(mailboxName, false)
		if err != nil {
			return fmt.Errorf("failed to select mailbox: %v", err)
		}

		seqSet := new(imap.SeqSet)
		seqSet.AddRange(1, mbox.Messages)
		items := []imap.FetchItem{imap.FetchUid}

		messages := make(chan *imap.Message, 10)
		go func() {
			if err := c.Fetch(seqSet, items, messages); err != nil {
				log.Println("Error fetching message UIDs:", err)
			}
		}()

		var newUIDs []uint32
		for msg := range messages {
			if _, exists := existingUIDSet[msg.Uid]; !exists {
				newUIDs = append(newUIDs, msg.Uid)
			}
		}

		if len(newUIDs) == 0 {
			log.Println("No new messages found.")
			return nil
		}

		seqSet = new(imap.SeqSet)
		seqSet.AddNum(newUIDs...)
		items = []imap.FetchItem{imap.FetchEnvelope, imap.FetchBodyStructure, imap.FetchUid}

		messages = make(chan *imap.Message, 10)
		go func() {
			if err := c.UidFetch(seqSet, items, messages); err != nil {
				log.Println("Error fetching new messages:", err)
			}
		}()

		strconv.ParseFloat("1.0", 64)

		for msg := range messages {
			email := mail.SerializableMessage{
				UID:         msg.Uid,
				Envelope:    msg.Envelope,
				MailboxName: mailboxName,
			}

			newMessages = append(newMessages, email)
		}

		return nil
	}

	if a.oauthToken != nil {
		oauthConfig := auth.GmailOAuthConfig
		_, err = mail.WithOAuthClient(a.imapUrl, a.emailAddr, a.oauthToken, oauthConfig, fetchMessages)
	} else {
		err = mail.WithClient(a.imapUrl, a.emailAddr, a.emailAppPassword, fetchMessages)
	}

	if err != nil {
		log.Println("Error fetching messages from server:", err)
		return
	}

	if len(newMessages) == 0 {
		log.Println("No new messages to update.")
		return
	}

	tx, err := a.db.Begin()
	if err != nil {
		log.Println("Error starting transaction to update messages:", err)
		return
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
        INSERT INTO messages (mailbox_name, uid, envelope, body_plain, body_html, body_raw, received_at, last_updated) 
        VALUES (?, ?, ?, ?, ?, ?, ?, ?)
    `)
	if err != nil {
		log.Println("Error preparing statement to insert messages:", err)
		return
	}
	defer stmt.Close()

	for _, msg := range newMessages {
		envelopeData, err := json.Marshal(msg.Envelope)
		if err != nil {
			log.Println("Error marshalling envelope for UID", msg.UID, ":", err)
			continue
		}

		_, err = stmt.Exec(mailboxName, msg.UID, envelopeData, msg.Body.Plain, msg.Body.HTML, nil, time.Now(), time.Now())
		if err != nil {
			log.Println("Error inserting message UID", msg.UID, "into database:", err)
		}
	}

	if err := tx.Commit(); err != nil {
		log.Println("Error committing transaction to update messages:", err)
		return
	}

	runtime.EventsEmit(a.ctx, "MessagesUpdated", mailboxName)
}

func fetchExistingUIDs(db *sql.DB, mailboxName string) ([]uint32, error) {
	rows, err := db.Query("SELECT uid FROM messages WHERE mailbox_name = ?", mailboxName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var uids []uint32
	for rows.Next() {
		var uid uint32
		if err := rows.Scan(&uid); err != nil {
			return nil, err
		}
		uids = append(uids, uid)
	}
	return uids, nil
}
