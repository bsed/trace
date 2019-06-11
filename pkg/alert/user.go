package alert

// User ....
type User struct {
	Mobile string
	Email  string
}

// NewUser ...
func NewUser() *User {
	return &User{}
}
