package mail

import (
	"bytes"
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
)

type EmailBody struct {
	Plain string `json:"plain"`
	HTML  string `json:"html"`
}

func FetchEmailBody(c *client.Client, uid uint32) (EmailBody, error) {
	log.Println("Fetching email body for UID:", uid)
	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uid)

	section := &imap.BodySectionName{}
	items := []imap.FetchItem{section.FetchItem()}

	messages := make(chan *imap.Message, 1)
	err := c.UidFetch(seqSet, items, messages)
	if err != nil {
		log.Printf("UidFetch error: %v\n", err)
		return EmailBody{}, err
	}

	msg := <-messages
	if msg == nil {
		log.Println("Server didn't return message")
		return EmailBody{}, fmt.Errorf("server didn't return message")
	}

	r := msg.GetBody(section)
	if r == nil {
		return EmailBody{}, fmt.Errorf("server didn't return message body")
	}

	emailMsg, err := mail.ReadMessage(r)
	if err != nil {
		log.Println("Error parsing email message:", err)
		return EmailBody{}, err
	}

	body, err := extractEmailBody(emailMsg)
	if err != nil {
		log.Println("Error extracting email body:", err)
		return EmailBody{}, err
	}

	return body, nil
}

func extractEmailBody(msg *mail.Message) (EmailBody, error) {
	contentType := msg.Header.Get("Content-Type")
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		log.Printf("Error parsing media type: %v\n", err)
		// Default to text/plain if Content-Type is missing or invalid
		mediaType = "text/plain"
	}

	log.Println("Extracting Email Body")

	var emailBody EmailBody

	if strings.HasPrefix(mediaType, "multipart/") {
		log.Println("Multipart message found: ", mediaType)

		boundary := params["boundary"]
		if boundary == "" {
			return EmailBody{}, fmt.Errorf("no boundary found for multipart message")
		}
		mr := multipart.NewReader(msg.Body, boundary)
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return EmailBody{}, err
			}

			partMediaType, _, _ := mime.ParseMediaType(p.Header.Get("Content-Type"))
			content, err := decodeEmailPart(p)
			if err != nil {
				continue
			}

			switch {
			case strings.Contains(partMediaType, "text/html"):
				emailBody.HTML = content
			case strings.Contains(partMediaType, "text/plain"):
				emailBody.Plain = content
			}
		}
	} else {
		// Single-part message
		content, err := decodeEmailPart(msg.Body)
		if err != nil {
			return EmailBody{}, err
		}
		if mediaType == "text/html" {
			emailBody.HTML = content
		} else {
			emailBody.Plain = content
		}
	}

	return emailBody, nil
}

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
