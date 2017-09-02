package discord

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/tchap/steemwatch/notifications/events"
)

func steemitLink(account string) string {
	return "@" + account
}

func steemdLink(account string) string {
	return "@" + account
}

// AccountUpdated

func renderAccountUpdatedEvent(event *events.AccountUpdated) string {
	return fmt.Sprintf(`
**-----**
Account update detected for %v.
`,
		steemitLink(event.Op.Account),
	)
}

// AccountWitnessVoted

func renderAccountWitnessVotedEvent(event *events.AccountWitnessVoted) string {
	var verb string
	if event.Op.Approve {
		verb = "approved"
	} else {
		verb = "unapproved"
	}

	return fmt.Sprintf(`
**-----**
%v %v witness %v.
`,
		steemitLink(event.Op.Account),
		verb,
		steemitLink(event.Op.Witness),
	)
}

// TransferMade

func renderTransferMadeEvent(event *events.TransferMade) string {
	op := event.Op
	if op.Memo != "" {
		return fmt.Sprintf(`
**-----**
%v transferred %v to %v using memo %v.
`,
			steemitLink(op.From),
			op.Amount,
			steemitLink(op.To),
			op.Memo,
		)
	}
	return fmt.Sprintf(
		"%v transferred %v to %v.",
		steemitLink(op.From),
		op.Amount,
		steemitLink(op.To),
	)
}

// UserMentioned

func renderUserMentionedEvent(event *events.UserMentioned) string {
	c := event.Content
	return fmt.Sprintf(`
**-----**
%v was mentioned by %v in https://steemit.com%v.
`,
		steemitLink(event.User),
		steemitLink(c.Author),
		c.URL,
	)
}

// UserFollowStatusChanged

func renderUserFollowStatusChangedEvent(event *events.UserFollowStatusChanged) string {
	op := event.Op

	follower := steemitLink(op.Follower)
	following := steemitLink(op.Following)

	var text string
	switch {
	case event.Followed():
		text = fmt.Sprintf("%v started following %v.", follower, following)
	case event.Muted():
		text = fmt.Sprintf("%v muted %v.", follower, following)
	case event.Reset():
		text = fmt.Sprintf("%v reset the follow status for %v.", follower, following)
	}

	return text
}

// StoryPublished

func renderStoryPublishedEvent(event *events.StoryPublished) string {
	c := event.Content

	return fmt.Sprintf(`
**-----**
%v has published or updated a story.

**Title:** %v
**Tags:** %v
**Link:** https://steemit.com%v
`,
		steemitLink(c.Author),
		c.Title,
		c.JsonMetadata.Tags,
		c.URL,
	)
}

// StoryVoted

func renderStoryVotedEvent(event *events.StoryVoted) string {
	o := event.Op
	c := event.Content

	return fmt.Sprintf(`
**-----**
%v cast a vote on a story by %v.

**Title:** %v
**Link:** https://steemit.com%v
**Vote weight:** %v
**Pending Payout:** %v
`,
		steemitLink(o.Voter),
		steemitLink(o.Author),
		c.Title,
		c.URL,
		o.Weight,
		c.PendingPayoutValue,
	)
}

// CommentPublished

func renderCommentPublishedEvent(event *events.CommentPublished) string {
	c := event.Content

	commentLines := make([]string, 0, 5)
	scanner := bufio.NewScanner(strings.NewReader(c.Body))
	for scanner.Scan() {
		commentLines = append(commentLines, scanner.Text())
	}

	extractLines := commentLines
	if len(extractLines) > 5 {
		extractLines = extractLines[:5]
	}

	extract := strings.Join(extractLines, "\n")
	if len(commentLines) > 5 {
		extract += fmt.Sprintf("\n<https://steemit.com%v|Read more...>", c.URL)
	}

	return fmt.Sprintf(`
**-----**
%v commented on @%v/%v.

**Content:** %v
`,
		steemitLink(c.Author),
		c.URL,
		c.ParentAuthor,
		c.ParentPermlink,
		extract,
	)
}

// CommentVoted

func renderCommentVotedEvent(event *events.CommentVoted) string {
	o := event.Op
	c := event.Content

	return fmt.Sprintf(`
**-----**
%v cast a vote on a comment @%v/%v.

**Link:** https://steemit.com%v
**Weight:** %v
**Pending Payout:** %v
`,
		steemitLink(o.Voter),
		c.Author,
		c.Permlink,
		c.URL,
		o.Weight,
		c.PendingPayoutValue,
	)
}
