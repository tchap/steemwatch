package facebook

import (
	"encoding/json"
	"net/http"

	"github.com/tchap/steemwatch/server/auth"

	"github.com/labstack/echo"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/facebook"
)

type FacebookProfile struct {
	Id    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

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
				"public_profile",
				"email",
			},
			Endpoint: facebook.Endpoint,
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

	// Get an authenticated HTTP client.
	httpClient := authenticator.config.Client(oauth2.NoContext, token)

	// Call Facebook API.
	resp, err := httpClient.Get("https://graph.facebook.com/v2.6/me?fields=name,email")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Decode the response.
	var me FacebookProfile
	if err := json.NewDecoder(resp.Body).Decode(&me); err != nil {
		return nil, err
	}

	// Make sure the email address is set.
	if me.Email == "" {
		return nil, errors.Errorf("Facebook did not return any email address: %+v", me)
	}

	// Assemble the profile that we use internally.
	return &auth.UserProfile{
		Email: me.Email,
	}, nil
}
