package slack

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/tchap/steemwatch/notifications/events"

	"github.com/pkg/errors"
)

//
// Slack webhook payload
//

type Payload struct {
	Text        string        `json:"text,omitempty"`
	Attachments []*Attachment `json:"attachments,omitempty"`
}

type Attachment struct {
	Fallback   string   `json:"fallback"`
	Color      string   `json:"color,omitempty"`
	Pretext    string   `json:"pretext,omitempty"`
	AuthorName string   `json:"author_name,omitempty"`
	AuthorLink string   `json:"author_link,omitempty"`
	AuthorIcon string   `json:"author_icon,omitempty"`
	Title      string   `json:"title,omitempty"`
	TitleLink  string   `json:"title_link,omitempty"`
	Text       string   `json:"text,omitempty"`
	Fields     []*Field `json:"fields,omitempty"`
	ImageURL   string   `json:"image_url,omitempty"`
	ThumbURL   string   `json:"thumb_url,omitempty"`
	Footer     string   `json:"footer,omitempty"`
	FooterIcon string   `json:"footer_icon,omitempty"`
	Timestamp  uint64   `json:"ts,omitempty"`
}

type Field struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short,omitempty"`
}

//
// Rendering
//

func makeMessage(attachment *Attachment) *Payload {
	return &Payload{
		Attachments: []*Attachment{attachment},
	}
}

// UserMentioned

func renderUserMentionedEvent(event *events.UserMentioned) (*Payload, error) {
	c := event.Content

	txt := fmt.Sprintf("@%v was <https://steemit.com%v|mentioned> by @%v in %v",
		event.User, c.URL, c.Author, c.Permlink)

	return &Payload{
		Text: txt,
	}, nil
}

// StoryPublished

func renderStoryPublishedEvent(event *events.StoryPublished) (*Payload, error) {
	c := event.Content
	r := bufio.NewReader(strings.NewReader(c.Body))

	summary, err := r.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			if summary == "" {
				summary = "<empty>"
			}
		} else {
			return nil, errors.Wrap(err, "failed to get story summary")
		}
	}

	return makeMessage(&Attachment{
		Fallback:  fmt.Sprintf(`@%v has published "%v".`, c.Author, c.Title),
		Color:     "#00C957",
		Pretext:   fmt.Sprintf("@%v has published or updated a story.", c.Author),
		Title:     c.Title,
		TitleLink: "https://steemit.com" + c.URL,
		Fields: []*Field{
			{
				Title: "Summary",
				Value: summary,
			},
			{
				Title: "Tags",
				Value: fmt.Sprintf("%v", c.JsonMetadata.Tags),
			},
		},
		ThumbURL: "https://steemit.com/images/favicons/favicon-96x96.png",
	}), nil
}

// StoryVoted

func renderStoryVotedEvent(event *events.StoryVoted) (*Payload, error) {
	o := event.Op
	c := event.Content

	evt := fmt.Sprintf("@%v cast a vote on a story by @%v.", o.Voter, o.Author)

	return makeMessage(&Attachment{
		Fallback:  evt,
		Color:     "#BDFCC9",
		Pretext:   evt,
		Title:     c.Title,
		TitleLink: "https://steemit.com" + c.URL,
		Fields: []*Field{
			{
				Title: "Vote Weight",
				Value: fmt.Sprintf("%v", o.Weight),
				Short: true,
			},
			{
				Title: "Story Pending Payout",
				Value: c.PendingPayoutValue,
				Short: true,
			},
		},
	}), nil
}

// CommentPublished

func renderCommentPublishedEvent(event *events.CommentPublished) (*Payload, error) {
	c := event.Content

	commentLines := make([]string, 0, 5)
	scanner := bufio.NewScanner(strings.NewReader(c.Body))
	for scanner.Scan() {
		commentLines = append(commentLines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to read comment body")
	}

	extractLines := commentLines
	if len(extractLines) > 5 {
		extractLines = extractLines[:5]
	}

	extract := strings.Join(extractLines, "\n")
	if len(commentLines) > 5 {
		extract += fmt.Sprintf("\n<https://steemit.com%v|Read more...>", c.URL)
	}

	evt := fmt.Sprintf("@%v commented on @%v/%v", c.Author, c.ParentAuthor, c.ParentPermlink)
	pre := fmt.Sprintf("@%v <https://steemit.com%v|commented> on @%v/%v",
		c.Author, c.URL, c.ParentAuthor, c.ParentPermlink)

	return makeMessage(&Attachment{
		Fallback: evt,
		Color:    "#FF9912",
		Pretext:  pre,
		Fields: []*Field{
			{
				Title: "Comment Body",
				Value: extract,
			},
		},
	}), nil
}

// CommentVoted

func renderCommentVotedEvent(event *events.CommentVoted) (*Payload, error) {
	o := event.Op
	c := event.Content

	evt := fmt.Sprintf("@%v cast a vote on comment @%v/%v", o.Voter, o.Author, o.Permlink)

	return makeMessage(&Attachment{
		Fallback:  evt,
		Color:     "#FFEBCD",
		Pretext:   evt,
		Title:     fmt.Sprintf("@%v/%v", c.Author, c.Permlink),
		TitleLink: "https://steemit.com" + c.URL,
		Fields: []*Field{
			{
				Title: "Vote Weight",
				Value: fmt.Sprintf("%v", o.Weight),
				Short: true,
			},
			{
				Title: "Comment Pending Payout",
				Value: c.PendingPayoutValue,
				Short: true,
			},
		},
	}), nil
}
