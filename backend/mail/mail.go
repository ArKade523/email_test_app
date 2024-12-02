package mail

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"net/textproto"
	"strings"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"golang.org/x/oauth2"
)

// SerializableMessage represents an email message that can be serialized
type SerializableMessage struct {
	SeqNum      uint32         `json:"seq_num"`
	Envelope    *imap.Envelope `json:"envelope"`
	Body        string         `json:"body"`
	MailboxName string         `json:"mailbox_name"`
}

const DEFAULT_EMAIL_COUNT = 10

// WithClient is a wrapper function that creates a new IMAP client and executes the provided function
func WithClient(imapUrl, emailAddr, emailAppPassword string, fn func(c *client.Client) error) error {
	c, err := GetClient(imapUrl, emailAddr, emailAppPassword)
	if err != nil {
		return err
	}
	defer c.Logout()
	return fn(c)
}

// Custom XOAUTH2 SASL client implementation
type XOAuth2Client struct {
	username    string
	accessToken string
}

func (a *XOAuth2Client) Start() (mech string, ir []byte, err error) {
	mech = "XOAUTH2"
	ir = []byte(fmt.Sprintf("user=%s\x00auth=Bearer %s\x01\x01", a.username, a.accessToken))
	return
}

func (a *XOAuth2Client) Next(challenge []byte) (response []byte, err error) {
	// No additional steps required
	return nil, nil
}

func WithOAuthClient(imapUrl, emailAddr string, token *oauth2.Token, oauthConfig *oauth2.Config, fn func(c *client.Client) error) (*oauth2.Token, error) {
	// Create a token source
	tokenSource := oauthConfig.TokenSource(context.Background(), token)

	// Obtain a new token (this will refresh the token if needed)
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, err
	}

	// Update the token if it has been refreshed
	if newToken.AccessToken != token.AccessToken {
		token = newToken
	}

	// Dial the IMAP server
	c, err := client.DialTLS(imapUrl, nil)
	if err != nil {
		return token, err
	}
	defer c.Logout()

	// Create the XOAUTH2 authentication mechanism
	auth := &XOAuth2Client{
		username:    emailAddr,
		accessToken: token.AccessToken,
	}

	// Authenticate
	if err := c.Authenticate(auth); err != nil {
		return token, err
	}

	// Execute the provided function
	err = fn(c)

	return token, err
}

// GetClient connects to the IMAP server and logs in
func GetClient(imapUrl string, emailAddr string, emailAppPassword string) (*client.Client, error) {
	c, err := client.DialTLS(imapUrl, nil)
	if err != nil {
		return nil, err
	}

	// Login
	if err := c.Login(emailAddr, emailAppPassword); err != nil {
		return nil, err
	}

	return c, nil
}

// FetchMailboxes fetches the list of mailboxes
func FetchMailboxes(c *client.Client) ([]*imap.MailboxInfo, error) {
	mboxes := make(chan *imap.MailboxInfo)
	var mailboxes []*imap.MailboxInfo

	// Use done channel to wait for the command to finish
	done := make(chan error, 1)

	go func() {
		// List mailboxes and send them to mboxes channel
		done <- c.List("", "*", mboxes)
	}()

	// Read from the mboxes channel until it's closed
	for mbox := range mboxes {
		mailboxes = append(mailboxes, mbox)
	}

	// Wait for the goroutine to finish and check for errors
	if err := <-done; err != nil {
		log.Println("Error fetching mailboxes:", err)
		return nil, err
	}

	return mailboxes, nil
}

// FetchEmailsForMailbox fetches the emails in the specified mailbox
func FetchEmailsForMailbox(c *client.Client, mailboxName string) ([]SerializableMessage, error) {
	// Select the mailbox
	mailbox, err := c.Select(mailboxName, false)
	if err != nil {
		return nil, err
	}

	if mailbox.Messages == 0 {
		log.Println("No messages in mailbox:", mailboxName)
		return nil, nil
	}

	from := uint32(1)
	to := mailbox.Messages
	if mailbox.Messages > DEFAULT_EMAIL_COUNT {
		from = mailbox.Messages - DEFAULT_EMAIL_COUNT - 1 // Adjusted to fetch the last 10 messages
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddRange(from, to)

	// Fetch only the envelope
	items := []imap.FetchItem{imap.FetchEnvelope}

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)

	go func() {
		done <- c.Fetch(seqSet, items, messages)
	}()

	var result []SerializableMessage
	for msg := range messages {
		if msg == nil || msg.Envelope == nil {
			continue
		}

		serializableMsg := SerializableMessage{
			SeqNum:      msg.SeqNum,
			Envelope:    msg.Envelope,
			Body:        "", // Body will be fetched later
			MailboxName: mailboxName,
		}
		// Prepend to get messages in descending order
		result = append([]SerializableMessage{serializableMsg}, result...)
	}

	if err := <-done; err != nil {
		return nil, err
	}

	return result, nil
}

// FetchEmailBody fetches the body of the specified email
func FetchEmailBody(c *client.Client, seqNum uint32) (string, error) {
	seqSet := new(imap.SeqSet)
	seqSet.AddNum(seqNum)

	// Fetch the BODY[] of the message
	section := &imap.BodySectionName{}
	items := []imap.FetchItem{section.FetchItem()}

	messages := make(chan *imap.Message, 1)
	done := make(chan error, 1)

	go func() {
		done <- c.Fetch(seqSet, items, messages)
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

	// Parse the email message using net/mail
	emailMsg, err := mail.ReadMessage(r)
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
func extractEmailBody(msg *mail.Message) (string, error) {
	contentType := msg.Header.Get("Content-Type")
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		// Default to text/plain if Content-Type is missing or invalid
		mediaType = "text/plain"
	}

	var body string

	if strings.HasPrefix(mediaType, "multipart/") {
		boundary := params["boundary"]
		if boundary == "" {
			return "", fmt.Errorf("no boundary found for multipart message")
		}
		mr := multipart.NewReader(msg.Body, boundary)
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return "", err
			}

			partMediaType, _, _ := mime.ParseMediaType(p.Header.Get("Content-Type"))
			content, err := decodeEmailPart(p)
			if err != nil {
				continue
			}

			if strings.Contains(partMediaType, "text/html") {
				body = content
				break // Prefer HTML content
			} else if strings.Contains(partMediaType, "text/plain") && body == "" {
				// Fallback to plain text if HTML is not found
				body = content
			}
		}
	} else {
		// Single-part message
		content, err := decodeEmailPart(msg.Body)
		if err != nil {
			return "", err
		}
		body = content
	}

	return body, nil
}

// decodeEmailPart decodes the content of an email part based on its encoding
func decodeEmailPart(part io.Reader) (string, error) {
	var buf bytes.Buffer
	_, err := buf.ReadFrom(part)
	if err != nil {
		return "", err
	}

	content := buf.Bytes()

	// Handle Content-Transfer-Encoding
	var decodedContent []byte

	encoding := "7bit" // Default encoding
	if header, ok := part.(interface{ Header() textproto.MIMEHeader }); ok {
		encoding = header.Header().Get("Content-Transfer-Encoding")
	}

	switch strings.ToLower(encoding) {
	case "base64":
		decodedContent, err = base64.StdEncoding.DecodeString(string(content))
		if err != nil {
			return "", err
		}
	case "quoted-printable":
		reader := quotedprintable.NewReader(bytes.NewReader(content))
		decodedContent, err = io.ReadAll(reader)
		if err != nil {
			return "", err
		}
	default:
		// 7bit, 8bit, binary, or unknown encodings
		decodedContent = content
	}

	return string(decodedContent), nil
}
