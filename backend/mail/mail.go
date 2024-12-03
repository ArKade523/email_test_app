package mail

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/quotedprintable"
	"slices"
	"strings"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message"
	"github.com/emersion/go-message/charset"
)

// SerializableMessage represents an email message that can be serialized
type SerializableMessage struct {
	UID         uint32         `json:"uid"`
	Envelope    *imap.Envelope `json:"envelope"`
	Body        string         `json:"body"`
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
			Body:        "",
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

// FetchEmailBody fetches the body of the specified email
func FetchEmailBody(c *client.Client, uid uint32) (string, error) {
	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uid)

	// Fetch the BODY[] of the message using UID
	section := &imap.BodySectionName{}
	items := []imap.FetchItem{section.FetchItem()}

	messages := make(chan *imap.Message, 1)
	done := make(chan error, 1)
	go func() {
		done <- c.UidFetch(seqSet, items, messages)
	}()

	msg := <-messages
	if err := <-done; err != nil {
		return "", err
	}

	if msg == nil {
		return "", fmt.Errorf("server didn't return message")
	}

	r := msg.GetBody(section)
	if r == nil {
		return "", fmt.Errorf("server didn't return message body")
	}

	// Wrap the reader with a larger buffer
	bufferedReader := bufio.NewReaderSize(r, 128*1024) // 128 KB buffer
	emailMsg, err := message.Read(bufferedReader)
	if err != nil {
		log.Println("Error parsing email message:", err)
		return "", err
	}

	// Extract the email body
	body, err := extractEmailBody(emailMsg)
	if err != nil {
		log.Println("Error extracting email body:", err)
		return "", err
	}

	return body, nil
}

// extractEmailBody extracts the email body from a mail.Message
func extractEmailBody(entity *message.Entity) (string, error) {
	mediaType, _, err := mime.ParseMediaType(entity.Header.Get("Content-Type"))
	if err != nil {
		mediaType = "text/plain"
	}

	var body string

	if strings.HasPrefix(mediaType, "multipart/") {
		mr := entity.MultipartReader()
		if mr == nil {
			return "", fmt.Errorf("error reading multipart message")
		}

		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Printf("Error reading multipart part: %v", err)
				continue
			}

			contentType := part.Header.Get("Content-Type")
			partMediaType, _, _ := mime.ParseMediaType(contentType)

			content, err := decodeEmailPart(part)
			if err != nil {
				log.Printf("Error decoding email part: %v", err)
				continue
			}

			if strings.Contains(partMediaType, "text/html") {
				body = content
				break // Prefer HTML content
			} else if strings.Contains(partMediaType, "text/plain") && body == "" {
				body = content
			}
		}
	} else {
		// Single-part message
		content, err := decodeEmailPart(entity)
		if err != nil {
			return "", err
		}
		body = content
	}

	return body, nil
}

// decodeEmailPart decodes the content of an email part based on its encoding
func decodeEmailPart(entity *message.Entity) (string, error) {
	var reader io.Reader = entity.Body

	encoding := entity.Header.Get("Content-Transfer-Encoding")
	switch strings.ToLower(encoding) {
	case "base64":
		reader = base64.NewDecoder(base64.StdEncoding, reader)
	case "quoted-printable":
		reader = quotedprintable.NewReader(reader)
	}

	decodedContent, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	return string(decodedContent), nil
}

func init() {
	message.CharsetReader = func(inputCharset string, input io.Reader) (io.Reader, error) {
		switch strings.ToLower(inputCharset) {
		case "utf-8", "us-ascii", "ascii":
			return input, nil
		default:
			log.Printf("Unknown charset: %s, falling back to UTF-8", inputCharset)
			return charset.Reader("utf-8", input)
		}
	}
}
