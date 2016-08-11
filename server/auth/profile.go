package auth

import (
	"github.com/tchap/steemwatch/server/users"
)

type SocialLink struct {
	ServiceName string
	UserKey     string
	UserName    string
}

type UserProfile struct {
	Email      string
	SocialLink *SocialLink
}

func (profile *UserProfile) AsUser() *users.User {
	user := &users.User{Email: profile.Email}

	if link := profile.SocialLink; link != nil {
		user.SocialLinks = map[string]*users.SocialLink{
			link.ServiceName: &users.SocialLink{
				UserKey:  link.UserKey,
				UserName: link.UserName,
			},
		}
	}

	return user
}
