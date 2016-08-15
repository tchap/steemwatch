package steemitchat

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/tchap/steemwatch/errs"
	"github.com/tchap/steemwatch/notifications/events"

	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
	"gopkg.in/mgo.v2/bson"
)

const DefaultMaxConcurrentRequests = 1000

const postMessageEndpointURL = "https://steemit.chat/api/v1/chat.postMessage"

//
// Settings
//

type Settings struct {
	Username string `bson:"username"`
}

func (settings *Settings) Validate() error {
	// Make sure the username is not empty.
	if settings.Username == "" {
		return errors.New("username is not set")
	}

	// Cool.
	return nil
}

func UnmarshalSettings(userId string, raw bson.Raw) (*Settings, error) {
	// Unmarshal.
	var settings Settings
	if err := raw.Unmarshal(&settings); err != nil {
		return nil, errors.Wrapf(
			err, "failed to unmarshal Steemit Chat settings for user %v", userId)
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
	daemonUserID    string
	daemonAuthToken string

	webhookTimeout        time.Duration
	maxConcurrentRequests uint
	requestSemaphore      chan struct{}
	termCh                chan struct{}
}

func NewNotifier(daemonUserID string, daemonAuthToken string, opts ...NotifierOption) *Notifier {
	notifier := &Notifier{
		daemonUserID:          daemonUserID,
		daemonAuthToken:       daemonAuthToken,
		webhookTimeout:        30 * time.Second,
		maxConcurrentRequests: DefaultMaxConcurrentRequests,
		termCh:                make(chan struct{}),
	}

	for _, opt := range opts {
		opt(notifier)
	}

	notifier.requestSemaphore = make(chan struct{}, notifier.maxConcurrentRequests)

	return notifier
}

type NotifierOption func(*Notifier)

func SetWebhookTimeout(timeout time.Duration) NotifierOption {
	return func(notifier *Notifier) {
		notifier.webhookTimeout = timeout
	}
}

func SetMaxConcurrentRequests(maxConcurrentRequests uint) NotifierOption {
	return func(notifier *Notifier) {
		notifier.maxConcurrentRequests = maxConcurrentRequests
	}
}

func (notifier *Notifier) DispatchAccountUpdatedEvent(
	userId string,
	userSettings bson.Raw,
	event *events.AccountUpdated,
) error {
	return notifier.dispatch(userId, userSettings, func() (*Payload, error) {
		return renderAccountUpdatedEvent(event)
	})
}

func (notifier *Notifier) DispatchTransferMadeEvent(
	userId string,
	userSettings bson.Raw,
	event *events.TransferMade,
) error {
	return notifier.dispatch(userId, userSettings, func() (*Payload, error) {
		return renderTransferMadeEvent(event)
	})
}

func (notifier *Notifier) DispatchUserMentionedEvent(
	userId string,
	userSettings bson.Raw,
	event *events.UserMentioned,
) error {
	return notifier.dispatch(userId, userSettings, func() (*Payload, error) {
		return renderUserMentionedEvent(event)
	})
}

func (notifier *Notifier) DispatchUserFollowStatusChangedEvent(
	userId string,
	userSettings bson.Raw,
	event *events.UserFollowStatusChanged,
) error {
	return notifier.dispatch(userId, userSettings, func() (*Payload, error) {
		return renderUserFollowStatusChangedEvent(event)
	})
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

	payload.Channel = "@" + settings.Username

	return notifier.send(payload)
}

func (notifier *Notifier) send(payload *Payload) error {
	// Acquire a request slot.
	select {
	case notifier.requestSemaphore <- struct{}{}:
		defer func() {
			<-notifier.requestSemaphore
		}()
	case <-notifier.termCh:
		return errs.ErrClosing
	}

	// Marshal the payload.
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(payload); err != nil {
		return errors.Wrap(err, "failed to encode Slack webhook")
	}

	// Send the webhook. Wait for the given timeout before cancelling.
	req := fasthttp.AcquireRequest()
	res := fasthttp.AcquireResponse()

	cleanup := func() {
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(res)
	}

	req.Header.SetMethod("POST")
	req.Header.SetContentType("application/json")
	req.Header.Set("X-User-Id", notifier.daemonUserID)
	req.Header.Set("X-Auth-Token", notifier.daemonAuthToken)
	req.SetRequestURI(postMessageEndpointURL)
	req.SetBodyStream(&body, body.Len())
	req.SetConnectionClose()

	if err := fasthttp.DoTimeout(req, res, notifier.webhookTimeout); err != nil {
		cleanup()
		return errors.Wrap(err, "failed to post Steemit Chat message")
	}

	if code := res.StatusCode(); code < 200 || code >= 300 {
		cleanup()
		return errors.Errorf("POST %v -> %v", postMessageEndpointURL, code)
	}

	cleanup()
	return nil
}

func (notifier *Notifier) Close() error {
	select {
	case <-notifier.termCh:
		return errs.ErrClosing
	default:
		close(notifier.termCh)
		return nil
	}
}
