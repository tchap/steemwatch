package google

import (
	"net/http"

	"github.com/tchap/steemwatch/server/auth"

	"github.com/labstack/echo"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/plus/v1"
)

type Authenticator struct {
	config *oauth2.Config
}

func NewAuthenticator(clientId, clientSecret, redirectURL string) *Authenticator {
	return &Authenticator{
		config: &oauth2.Config{
			ClientID:     clientId,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes: []string{
				plus.UserinfoProfileScope,
				plus.UserinfoEmailScope,
			},
			Endpoint: google.Endpoint,
		},
	}
}

func (authenticator *Authenticator) Authenticate(ctx echo.Context) error {
	// Redirect to the consent page.
	consentPageURL := authenticator.config.AuthCodeURL("state")
	return ctx.Redirect(http.StatusTemporaryRedirect, consentPageURL)
}

func (authenticator *Authenticator) Callback(ctx echo.Context) (*auth.UserProfile, error) {
	// Handle the exchange code to initiate a transport.
	token, err := authenticator.config.Exchange(oauth2.NoContext, ctx.QueryParam("code"))
	if err != nil {
		return nil, err
	}

	// Get Google Plus API client.
	client := authenticator.config.Client(oauth2.NoContext, token)
	service, err := plus.New(client)
	if err != nil {
		return nil, err
	}

	// Call Google API to get the user profile.
	call := service.People.Get("me")
	me, err := call.Do()
	if err != nil {
		return nil, err
	}

	// Get the account email.
	var email string
	for _, personEmail := range me.Emails {
		if personEmail.Type == "account" {
			email = personEmail.Value
		}
	}

	// Assemble the profile that we use internally.
	return &auth.UserProfile{
		Email: email,
	}, nil
}
