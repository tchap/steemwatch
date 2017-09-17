package github

import (
	"errors"
	"net/http"

	"github.com/tchap/steemwatch/server/auth"

	"github.com/google/go-github/github"
	"github.com/labstack/echo"
	"golang.org/x/oauth2"
	githubAuth "golang.org/x/oauth2/github"
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
				"user",
			},
			Endpoint: githubAuth.Endpoint,
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

	// Get a GitHub client.
	client := github.NewClient(httpClient)

	// Collect data from the GitHub API.
	errCh := make(chan error, 2)

	reqCtx := ctx.Request().Context()
	var me *github.User
	go func() {
		user, _, err := client.Users.Get(reqCtx, "")
		if err != nil {
			errCh <- err
			return
		}
		me = user
		errCh <- nil
	}()

	var emails []*github.UserEmail
	go func() {
		userEmails, _, err := client.Users.ListEmails(reqCtx, nil)
		if err != nil {
			errCh <- err
		}
		emails = userEmails
		errCh <- nil
	}()

	for i := 0; i < cap(errCh); i++ {
		if err := <-errCh; err != nil {
			return nil, err
		}
	}

	var email string
	for _, userEmail := range emails {
		if *userEmail.Primary && *userEmail.Verified {
			email = *userEmail.Email
		}
	}
	if email == "" {
		return nil, errors.New("GitHub auth: no verified email address found")
	}

	// Assemble the profile that we use internally.
	return &auth.UserProfile{
		Email: email,
	}, nil
}
