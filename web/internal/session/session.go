package session

import "sync"

var Sessions = &sync.Map{}
var UsersMap = &sync.Map{}
var UsersList = make([]*User, 0)
