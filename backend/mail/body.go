package mail

import (
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"strings"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message"
	"github.com/emersion/go-message/charset"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
)

type EmailBody struct {
	Plain string `json:"plain"`
	HTML  string `json:"html"`
}

// FetchEmailBody fetches the body of the specified email
func FetchEmailBody(c *client.Client, uid uint32) (EmailBody, error) {
	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uid)

	section := &imap.BodySectionName{}
	items := []imap.FetchItem{section.FetchItem()}

	messages := make(chan *imap.Message, 1)
	err := c.UidFetch(seqSet, items, messages)
	if err != nil {
		return EmailBody{}, err
	}

	msg := <-messages
	if msg == nil {
		return EmailBody{}, fmt.Errorf("server didn't return message")
	}

	r := msg.GetBody(section)
	if r == nil {
		return EmailBody{}, fmt.Errorf("server didn't return message body")
	}

	emailMsg, err := message.Read(r)
	if err != nil {
		return EmailBody{}, fmt.Errorf("error parsing email message: %v with uid %d", err, uid)
	}

	body, err := extractEmailBody(emailMsg)
	if err != nil {
		return EmailBody{}, fmt.Errorf("error extracting email body: %v", err)
	}

	return body, nil
}

// extractEmailBody extracts the email body from a message.Entity
func extractEmailBody(entity *message.Entity) (EmailBody, error) {
	mediaType, params, err := mime.ParseMediaType(entity.Header.Get("Content-Type"))
	if err != nil {
		return EmailBody{}, fmt.Errorf("error parsing media type: %v", err)
	}

	var emailBody EmailBody

	if strings.HasPrefix(mediaType, "multipart/") {
		boundary, ok := params["boundary"]
		if !ok {
			return EmailBody{}, fmt.Errorf("missing boundary in multipart email")
		}

		mr := multipart.NewReader(entity.Body, boundary)

		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return EmailBody{}, fmt.Errorf("error reading part: %v", err)
			}

			partMediaType, _, err := mime.ParseMediaType(part.Header.Get("Content-Type"))
			if err != nil {
				return EmailBody{}, fmt.Errorf("error parsing part media type: %v", err)
			}

			partBody, err := io.ReadAll(part)
			if err != nil {
				return EmailBody{}, fmt.Errorf("error reading part body: %v. media type: %s", err, partMediaType)
			}

			switch partMediaType {
			case "text/plain":
				emailBody.Plain = string(partBody)
			case "text/html":
				emailBody.HTML = string(partBody)
			}
		}
	} else {
		rawBody, err := io.ReadAll(entity.Body)
		if err != nil {
			return EmailBody{}, fmt.Errorf("error reading email body: %v", err)
		}

		switch mediaType {
		case "text/plain":
			emailBody.Plain = string(rawBody)
		case "text/html":
			emailBody.HTML = string(rawBody)
		}
	}

	return emailBody, nil
}

func init() {
	charset.RegisterEncoding("iso-8859-1", charmap.ISO8859_1)
	charset.RegisterEncoding("windows-1252", charmap.Windows1252)
	charset.RegisterEncoding("windows-1251", charmap.Windows1251)
	charset.RegisterEncoding("ascii", encoding.Nop)
}
