package mail

import (
	"log"
	"slices"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

// SerializableMessage represents an email message that can be serialized
type SerializableMessage struct {
	UID         uint32         `json:"uid"`
	Envelope    *imap.Envelope `json:"envelope"`
	Body        EmailBody      `json:"body"`
	MailboxName string         `json:"mailbox_name"`
}

const DEFAULT_EMAIL_COUNT = 10

// FetchMailboxes fetches the list of mailboxes
func FetchMailboxes(c *client.Client) ([]*imap.MailboxInfo, error) {
	var mailboxes []*imap.MailboxInfo
	mboxes := make(chan *imap.MailboxInfo)

	// List mailboxes synchronously
	done := make(chan error, 1)
	go func() {
		done <- c.List("", "*", mboxes)
	}()

	for mbox := range mboxes {
		mailboxes = append(mailboxes, mbox)
	}

	if err := <-done; err != nil {
		log.Println("Error fetching mailboxes:", err)
		return nil, err
	}

	return mailboxes, nil
}

// FetchEmailsForMailbox fetches the emails in the specified mailbox
func FetchEmailsForMailbox(c *client.Client, mailboxName string, start, limit uint32) ([]SerializableMessage, error) {
	// Select the mailbox
	_, err := c.Select(mailboxName, false)
	if err != nil {
		return nil, err
	}

	// Search for all message UIDs
	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{imap.DeletedFlag}
	uids, err := c.UidSearch(criteria)
	if err != nil {
		return nil, err
	}

	totalMessages := uint32(len(uids))
	if totalMessages == 0 {
		log.Println("No messages in mailbox:", mailboxName)
		return nil, nil
	}

	// Reverse the UIDs to have the most recent emails first
	for i, j := 0, len(uids)-1; i < j; i, j = i+1, j-1 {
		uids[i], uids[j] = uids[j], uids[i]
	}

	// Adjust start and end indices
	if start >= totalMessages {
		return nil, nil // No more messages to fetch
	}
	end := start + limit
	if end > totalMessages {
		end = totalMessages
	}

	// Get the subset of UIDs to fetch
	uidsToFetch := uids[start:end]

	// Create a sequence set with the UIDs to fetch
	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uidsToFetch...)

	// Fetch envelopes using UIDs
	items := []imap.FetchItem{imap.FetchEnvelope, imap.FetchUid}
	messages := make(chan *imap.Message, len(uidsToFetch))
	done := make(chan error, 1)
	go func() {
		done <- c.UidFetch(seqSet, items, messages)
	}()

	var result []SerializableMessage
	for msg := range messages {
		if msg == nil || msg.Envelope == nil {
			continue
		}

		serializableMsg := SerializableMessage{
			UID:         msg.Uid,
			Envelope:    msg.Envelope,
			Body:        EmailBody{},
			MailboxName: mailboxName,
		}
		result = append(result, serializableMsg)
	}

	if err := <-done; err != nil {
		return nil, err
	}

	slices.Reverse(result)

	return result, nil
}
