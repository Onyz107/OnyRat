package auth

import "sync"

// Constant for the authorization stream name
const authStream = "authorizationStream"

var bufPool = sync.Pool{
	New: func() any {
		return make([]byte, 16*1024)
	},
}
