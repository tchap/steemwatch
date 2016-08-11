package users

type SocialLink struct {
	UserKey  string
	UserName string
}

type User struct {
	Id          string
	Email       string
	SocialLinks map[string]*SocialLink
}
