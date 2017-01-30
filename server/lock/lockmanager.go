package lock

import (
	"dfs/comm"
	"strings"
	"sync"
)

var (
	ErrorResourceIsLocked = "Resource is locked."
)

type LockManager struct {
	mutex              sync.Mutex
	clock              int64
	lockMap            map[string]int
	awaitingPermission map[string]chan bool

	msgHub *comm.MessageHub
}

func (lm *LockManager) HandleMessage(msg comm.Message) {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()
	switch msg.Type {
	case comm.MessageTypeRequestLock:
		var request comm.MessageRequestLock
		if _, exists := lm.awaitingPermission[request.Resource]; !exists {
		} else {
		}
	case comm.MessageTypeGrantLockPermission:
	}
}

func (lm *LockManager) LockResource(resource string) error {
	msg := comm.Message{Type: comm.MessageTypeRequestLock}
	requestMsg := comm.MessageRequestLock{
		Resource: resource,
		Clock:    lm.clock,
	}

	msg.EncodeData(requestMsg)

	lm.mutex.Lock()
	if _, exists := lm.lockMap[resource]; exists {
		return ErrorResourceIsLocked
	}
	waitChan := make(chan bool, 1)
	lm.awaitingPermission[resource] = waitChan
	lm.mutex.Unlock()

	lm.msgHub.Broadcast(msg)

	granted := <-waitChan

	if !granted {
		return ErrorResourceIsLocked
	}

	return nil
}

func (lm *LockManager) UnlockResource(resource string) error {
	return nil
}
