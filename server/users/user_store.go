package users

// Store takes care of mapping between plaintext session cookie values and user profiles.
type Store interface {
	LoadUser(sessionCookie string) (user *User, err error)
	StoreUser(user *User) (sessionCookie string, err error)
}
