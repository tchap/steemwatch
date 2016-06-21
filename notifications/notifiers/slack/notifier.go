package slack

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/tchap/steemwatch/notifications/events"

	"github.com/pkg/errors"
	"gopkg.in/mgo.v2/bson"
)

//
// Settings
//

type Settings struct {
	WebhookURL string `bson:"webhookURL"`
}

func (settings *Settings) Validate() error {
	// Make sure the webhook URL is not empty.
	if settings.WebhookURL == "" {
		return errors.New("webhookURL is not set")
	}

	// Make the webhook URL is a valid URL.
	if _, err := url.Parse(settings.WebhookURL); err != nil {
		return errors.New("webhookURL is not a valid URL")
	}

	// Cool.
	return nil
}

func UnmarshalSettings(userId string, raw bson.Raw) (*Settings, error) {
	// Unmarshal.
	var settings Settings
	if err := raw.Unmarshal(&settings); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal Slack settings for user %v", userId)
	}

	// Validate.
	if err := settings.Validate(); err != nil {
		return nil, err
	}

	// Cool.
	return &settings, nil
}

//
// Notifier
//

type Notifier struct {
	webhookTimeout time.Duration
}

func NewNotifier(opts ...NotifierOption) *Notifier {
	notifier := &Notifier{
		webhookTimeout: 30 * time.Second,
	}

	for _, opt := range opts {
		opt(notifier)
	}

	return notifier
}

type NotifierOption func(*Notifier)

func SetWebhookTimeout(timeout time.Duration) NotifierOption {
	return func(notifier *Notifier) {
		notifier.webhookTimeout = timeout
	}
}

func (notifier *Notifier) DispatchStoryPublishedEvent(
	userId string,
	userSettings bson.Raw,
	event *events.StoryPublished,
) error {
	return notifier.dispatch(userId, userSettings, func() (*Payload, error) {
		return renderStoryPublishedEvent(event)
	})
}

func (notifier *Notifier) DispatchStoryVotedEvent(
	userId string,
	userSettings bson.Raw,
	event *events.StoryVoted,
) error {
	return notifier.dispatch(userId, userSettings, func() (*Payload, error) {
		return renderStoryVotedEvent(event)
	})
}

func (notifier *Notifier) DispatchCommentPublishedEvent(
	userId string,
	userSettings bson.Raw,
	event *events.CommentPublished,
) error {
	return notifier.dispatch(userId, userSettings, func() (*Payload, error) {
		return renderCommentPublishedEvent(event)
	})
}

func (notifier *Notifier) DispatchCommentVotedEvent(
	userId string,
	userSettings bson.Raw,
	event *events.CommentVoted,
) error {
	return notifier.dispatch(userId, userSettings, func() (*Payload, error) {
		return renderCommentVotedEvent(event)
	})
}

func (notifier *Notifier) dispatch(
	userId string,
	userSettings bson.Raw,
	render func() (*Payload, error),
) error {
	settings, err := UnmarshalSettings(userId, userSettings)
	if err != nil {
		return err
	}

	payload, err := render()
	if err != nil {
		return err
	}

	return notifier.send(settings.WebhookURL, payload)
}

func (notifier *Notifier) send(webhookURL string, payload *Payload) error {
	// Marshal the payload.
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(payload); err != nil {
		return errors.Wrap(err, "failed to encode Slack webhook")
	}

	// Send the webhook. Wait for the given timeout before cancelling.
	req, err := http.NewRequest("POST", webhookURL, &body)
	if err != nil {
		return errors.Wrap(err, "failed to create webhook request")
	}
	req.Header.Set("Content-Type", "application/json")

	transport := &http.Transport{}
	client := &http.Client{Transport: transport}

	errCh := make(chan error, 1)
	go func() {
		_, err := client.Do(req)
		if err != nil {
			err = errors.Wrap(err, "failed to send Slack webhook")
		}
		errCh <- err
	}()

	select {
	case err := <-errCh:
		return err
	case <-time.After(notifier.webhookTimeout):
		transport.CancelRequest(req)
		return <-errCh
	}
}
