package mail

import (
	"context"
	"errors"
	"fmt"

	"github.com/emersion/go-imap/client"
	"golang.org/x/oauth2"
)

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
	done        bool
}

func (a *XOAuth2Client) Start() (mech string, ir []byte, err error) {
	mech = "XOAUTH2"
	ir = []byte(fmt.Sprintf("user=%s\x01auth=Bearer %s\x01\x01", a.username, a.accessToken))
	return
}

func (a *XOAuth2Client) Next(challenge []byte) (response []byte, err error) {
	if a.done {
		return nil, errors.New("unexpected server challenge")
	}
	a.done = true
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
