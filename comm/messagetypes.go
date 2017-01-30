package comm

import (
	"fmt"
)

type MessageNodeStatus struct {
	RequestsPerMinute int
	TokenCount        int
	RequestCounter    int
}

func (msg MessageNodeStatus) String() string {
	return fmt.Sprintf("RPM: %d\nTC: %d\nRC: %d\n",
		msg.RequestsPerMinute,
		msg.TokenCount,
		msg.RequestCounter)
}

type MessageRequestToken struct {
	Path string
}

type MessageToken struct {
	Path  string
	Token string
}

type MessageRequestLock struct {
	Resource string
	Clock    int64
}

type MessageGrantLockPermission struct {
	Resource string
}
