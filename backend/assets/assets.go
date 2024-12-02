package assets

import (
	"embed"
)

//go:embed oauth_success.html
var OauthSuccessHTML embed.FS

//go:embed appicon.png
var AppIconPNG []byte
