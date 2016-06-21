package auth

import (
	"github.com/tchap/steemwatch/server/users"
)

type UserProfile struct {
	Email string
}

func (profile *UserProfile) AsUser() *users.User {
	return &users.User{Email: profile.Email}
}
