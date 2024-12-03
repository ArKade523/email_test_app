package wails_app

import (
	"email_test_app/backend/mail"
	"encoding/json"
	"log"
)

func (a *App) GetMailboxes() []string {
	rows, err := a.db.Query("SELECT name FROM mailboxes")
	if err != nil {
		log.Println("Error querying mailboxes from database:", err)
		return nil
	}
	defer rows.Close()

	var mailboxes []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			log.Println("Error scanning mailbox row:", err)
			continue
		}
		mailboxes = append(mailboxes, name)
	}

	return mailboxes
}

// GetEmailsForMailbox returns emails for a mailbox, using cache if available
func (a *App) GetEmailsForMailbox(mailboxName string, start, limit uint32) []mail.SerializableMessage {
	rows, err := a.db.Query(`
        SELECT uid, envelope FROM messages 
        WHERE mailbox_name = ? 
        ORDER BY received_at DESC 
        LIMIT ? OFFSET ?`, mailboxName, limit, start)
	if err != nil {
		log.Println("Error querying messages from database:", err)
		return nil
	}
	defer rows.Close()

	var messages []mail.SerializableMessage
	for rows.Next() {
		var msg mail.SerializableMessage
		var envelopeData []byte
		if err := rows.Scan(&msg.UID, &envelopeData); err != nil {
			log.Println("Error scanning message row:", err)
			continue
		}

		// Deserialize the envelope
		if err := json.Unmarshal(envelopeData, &msg.Envelope); err != nil {
			log.Println("Error unmarshalling envelope:", err)
			continue
		}

		msg.MailboxName = mailboxName
		msg.Body = "" // Ensure the body is not sent to the frontend
		messages = append(messages, msg)
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

	rows, err := a.db.Query(`
		SELECT body FROM messages
		WHERE mailbox_name = ? AND uid = ?
		LIMIT 1
	`, mailboxName, seqNum)

	if err != nil {
		log.Println("Error querying email body from database:", err)
		return ""
	}
	defer rows.Close()

	var body string
	if rows.Next() {
		if err := rows.Scan(&body); err != nil {
			log.Println("Error scanning email body row:", err)
			return ""
		}
	}

	return body
}
