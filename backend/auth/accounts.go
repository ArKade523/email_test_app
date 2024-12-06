package auth

import "time"

type Account struct {
	Id                  int64  `json:"id"`
	Email               string `json:"email"`
	ImapUrl             string `json:"imap_url"`
	OAuthAccessToken    string `json:"oauth_access_token"`
	OAuthRefreshToken   string `json:"oauth_refresh_token"`
	OAuthExpiry         int64  `json:"oauth_expiry"`
	AppSpecificPassword string `json:"app_specific_password"`
}

func (a *Account) IsOAuthExpired() bool {
	return a.OAuthExpiry < 0 || a.OAuthExpiry < time.Now().Unix()
}

func (a *Account) IsOAuthValid() bool {
	return a.OAuthAccessToken != "" && !a.IsOAuthExpired()
}

func (a *Account) IsAppSpecificPasswordValid() bool {
	return a.AppSpecificPassword != ""
}
