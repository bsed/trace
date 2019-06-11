package control

import (
	"sync"

	"github.com/bsed/trace/pkg/alert"
)

// Users  users
type Users struct {
	sync.RWMutex
	Users map[string]*alert.User
}

func (u *Users) add(id, email, mobile string) {
	u.RLock()
	user, ok := u.Users[id]
	u.RUnlock()
	if !ok {
		user = alert.NewUser()
		u.Users[id] = user
	}
	user.Email = email
	user.Mobile = mobile
}

func (u *Users) get(id string) (*alert.User, bool) {
	u.RLock()
	user, ok := u.Users[id]
	u.RUnlock()
	return user, ok
}

// newUsers ...
func newUsers() *Users {
	return &Users{
		Users: make(map[string]*alert.User),
	}
}
