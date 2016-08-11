package reddit

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tchap/steemwatch/server/auth"

	"github.com/labstack/echo"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

const StateCookieName = "reddit_oauth2_state"

const UserAgent = "SteemWatch"

type Authenticator struct {
	config   *oauth2.Config
	forceSSL bool
}

func NewAuthenticator(clientID, clientSecret, redirectURL string, forceSSL bool) *Authenticator {
	return &Authenticator{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes: []string{
				"identity",
			},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://www.reddit.com/api/v1/authorize",
				TokenURL: "https://www.reddit.com/api/v1/access_token",
			},
		},
		forceSSL: forceSSL,
	}
}

func (authenticator *Authenticator) Authenticate(ctx echo.Context) error {
	// Generate random state.
	state, err := generateState()
	if err != nil {
		return err
	}

	// Store the state in the state cookie.
	cookie := &echo.Cookie{}
	cookie.SetName(StateCookieName)
	cookie.SetValue(state)
	cookie.SetHTTPOnly(true)
	cookie.SetSecure(authenticator.forceSSL)

	ctx.SetCookie(cookie)

	// Redirect to the consent page.
	v := url.Values{
		"client_id":     {authenticator.config.ClientID},
		"redirect_uri":  {authenticator.config.RedirectURL},
		"response_type": {"code"},
		"scope":         {"identity"},
		"state":         {state},
	}
	consentPageURL := authenticator.config.Endpoint.AuthURL + "?" + v.Encode()
	return ctx.Redirect(http.StatusTemporaryRedirect, consentPageURL)
}

func (authenticator *Authenticator) Callback(ctx echo.Context) (*auth.UserProfile, error) {
	// Get the OAuth2 state cookie.
	stateCookie, err := ctx.Cookie(StateCookieName)
	if err != nil {
		return nil, errors.Wrap(err, "reddit: failed to get state cookie")
	}

	// Clear the cookie.
	cookie := &echo.Cookie{}
	cookie.SetName(StateCookieName)
	cookie.SetValue("unset")
	cookie.SetHTTPOnly(true)
	cookie.SetSecure(authenticator.forceSSL)
	cookie.SetExpires(time.Now().Add(-24 * time.Hour))

	ctx.SetCookie(cookie)

	// Make sure the query param matches the state cookie.
	state := ctx.QueryParam("state")
	if v := stateCookie.Value(); v != state {
		return nil, errors.Errorf("reddit: state mismatch: %v != %v", v, state)
	}

	// Get the access token.
	token, err := authenticator.getAccessToken(ctx.QueryParam("code"))
	if err != nil {
		return nil, err
	}

	// Get an authenticated HTTP client.
	httpClient := authenticator.config.Client(oauth2.NoContext, token)

	// Call Reddit API to get the current user's profile.
	req, err := http.NewRequest("GET", "https://oauth.reddit.com/api/v1/me", nil)
	if err != nil {
		return nil, errors.Wrap(err, "reddit: failed to create profile request")
	}
	req.Header.Set("User-Agent", UserAgent)

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "reddit: failed to get Reddit profile")
	}
	defer res.Body.Close()

	// Read the response body.
	body, err := ioutil.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, errors.Wrap(err, "reddit: failed to read profile body")
	}

	// Unmarshal the response body.
	var profile struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(body, &profile); err != nil {
		return nil, errors.Wrap(err, "reddit: failed to unmarshal profile")
	}

	// At last, return the normalized profile.
	return &auth.UserProfile{
		SocialLink: &auth.SocialLink{
			ServiceName: "reddit",
			UserKey:     profile.Name,
			UserName:    profile.Name,
		},
	}, nil
}

func (authenticator *Authenticator) getAccessToken(code string) (*oauth2.Token, error) {
	config := authenticator.config

	v := url.Values{
		"client_id":     {config.ClientID},
		"client_secret": {config.ClientSecret},
		"redirect_uri":  {config.RedirectURL},
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"scope":         {"identity"},
	}

	req, err := http.NewRequest("POST", config.Endpoint.TokenURL, strings.NewReader(v.Encode()))
	if err != nil {
		return nil, errors.Wrap(err, "reddit: failed to create token request")
	}

	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(config.ClientID, config.ClientSecret)
	req.Close = true

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "reddit: failed to get access token")
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, errors.Wrap(err, "reddit: cannot fetch token")
	}

	if code := res.StatusCode; code < 200 || code >= 300 {
		return nil, errors.Wrapf(
			err, "reddit: cannot fetch token\nResponse: %s", res.Status, body)
	}

	// Unmarshal the access token.
	var tokenRaw struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tokenRaw); err != nil {
		return nil, errors.Wrap(err, "reddit: failed to unmarshal access token")
	}

	return &oauth2.Token{
		AccessToken: tokenRaw.AccessToken,
		TokenType:   tokenRaw.TokenType,
		Expiry:      time.Now().Add(time.Duration(tokenRaw.ExpiresIn) * time.Second),
	}, nil
}

func generateState() (string, error) {
	raw := make([]byte, 258/8)
	if _, err := rand.Read(raw); err != nil {
		return "", errors.Wrap(err, "failed to generate OAuth2 state")
	}
	return base64.StdEncoding.EncodeToString(raw), nil
}
