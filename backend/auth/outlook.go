package auth

import (
	"golang.org/x/oauth2"
)

var OutlookOAuthConfig = &oauth2.Config{
	ClientID:    "YOUR_MICROSOFT_CLIENT_ID",
	RedirectURL: "http://localhost:PORT/oauth2callback",
	Scopes:      []string{"https://outlook.office.com/IMAP.AccessAsUser.All"},
	Endpoint: oauth2.Endpoint{
		AuthURL:  "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
		TokenURL: "https://login.microsoftonline.com/common/oauth2/v2.0/token",
	},
}
