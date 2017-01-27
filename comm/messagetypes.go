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
