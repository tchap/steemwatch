package eventstream

import (
	"bufio"
	"strings"

	"github.com/tchap/steemwatch/notifications/events"
)

type Event struct {
	Kind    string      `json:"kind"`
	Payload interface{} `json:"payload,omitempty"`
}

type AccountUpdatedPayload struct {
	Account string `json:"account"`
}

func formatAccountUpdated(event *events.AccountUpdated) interface{} {
	return &Event{
		Kind: "account.updated",
		Payload: &AccountUpdatedPayload{
			Account: event.Op.Account,
		},
	}
}

type TransferMadePayload struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Amount string `json:"amount"`
	Memo   string `json:"memo,omitempty"`
}

func formatTransferMade(event *events.TransferMade) interface{} {
	return &Event{
		Kind: "transfer.made",
		Payload: &TransferMadePayload{
			From:   event.Op.From,
			To:     event.Op.To,
			Amount: event.Op.Amount,
			Memo:   event.Op.Memo,
		},
	}
}

type UserMentionedPayload struct {
	User     string `json:"user"`
	URL      string `json:"url"`
	Author   string `json:"author"`
	Permlink string `json:"permlink"`
}

func formatUserMentioned(event *events.UserMentioned) interface{} {
	return &Event{
		Kind: "user.mentioned",
		Payload: &UserMentionedPayload{
			User:     event.User,
			URL:      event.Content.URL,
			Author:   event.Content.Author,
			Permlink: event.Content.Permlink,
		},
	}
}

type StoryPublishedPayload struct {
	Author string   `json:"author"`
	Title  string   `json:"title"`
	URL    string   `json:"url"`
	Tags   []string `json:"tags"`
}

func formatStoryPublished(event *events.StoryPublished) interface{} {
	return &Event{
		Kind: "story.published",
		Payload: &StoryPublishedPayload{
			Author: event.Content.Author,
			Title:  event.Content.Title,
			URL:    event.Content.URL,
			Tags:   event.Content.JsonMetadata.Tags,
		},
	}
}

type StoryVotedPayload struct {
	Voter              string `json:"voter"`
	VoteWeight         string `json:"voteWeight"`
	Author             string `json:"author"`
	Title              string `json:"title"`
	URL                string `json:"url"`
	TotalPayout        string `json:"totalPayout"`
	PendingPayout      string `json:"pendingPayout"`
	TotalPendingPayout string `json:"totalPendingPayout"`
}

func formatStoryVoted(event *events.StoryVoted) interface{} {
	return &Event{
		Kind: "story.voted",
		Payload: &StoryVotedPayload{
			Voter:              event.Op.Voter,
			VoteWeight:         event.Op.Weight.String(),
			Author:             event.Content.Author,
			Title:              event.Content.Title,
			URL:                event.Content.URL,
			TotalPayout:        event.Content.TotalPayoutValue,
			PendingPayout:      event.Content.PendingPayoutValue,
			TotalPendingPayout: event.Content.TotalPendingPayoutValue,
		},
	}
}

type CommentPublishedPayload struct {
	Author         string `json:"author"`
	URL            string `json:"url"`
	ParentAuthor   string `json:"parentAuthor"`
	ParentPermlink string `json:"parentPermlink"`
	Content        string `json:"content,omitempty"`
	ReadMore       bool   `json:"more,omitempty"`
}

func formatCommentPublished(event *events.CommentPublished) interface{} {
	commentLines := make([]string, 0, 5)
	scanner := bufio.NewScanner(strings.NewReader(event.Content.Body))
	i := 0
	for ; scanner.Scan() && i < 5; i++ {
		commentLines = append(commentLines, scanner.Text())
	}
	// We ignore the potential error here.

	content := strings.Join(commentLines[:5], "\n")
	more := i == 5

	return &Event{
		Kind: "comment.published",
		Payload: &CommentPublishedPayload{
			Author:         event.Content.Author,
			URL:            event.Content.URL,
			ParentAuthor:   event.Content.ParentAuthor,
			ParentPermlink: event.Content.ParentPermlink,
			Content:        content,
			ReadMore:       more,
		},
	}
}

type CommentVotedPayload struct {
	Voter              string `json:"voter"`
	VoteWeight         string `json:"voteWeight"`
	Author             string `json:"author"`
	Permlink           string `json:"permlink"`
	URL                string `json:"url"`
	TotalPayout        string `json:"totalPayout"`
	PendingPayout      string `json:"pendingPayout"`
	TotalPendingPayout string `json:"totalPendingPayout"`
}

func formatCommentVoted(event *events.CommentVoted) interface{} {
	return &Event{
		Kind: "comment.voted",
		Payload: &CommentVotedPayload{
			Voter:              event.Op.Voter,
			VoteWeight:         event.Op.Weight.String(),
			Author:             event.Content.Author,
			Permlink:           event.Content.Permlink,
			URL:                event.Content.URL,
			TotalPayout:        event.Content.TotalPayoutValue,
			PendingPayout:      event.Content.PendingPayoutValue,
			TotalPendingPayout: event.Content.TotalPendingPayoutValue,
		},
	}
}
