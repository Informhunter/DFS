package lock

import (
	"dfs/comm"
	"dfs/server/node"
	"errors"
	"fmt"
	"strings"
	"sync"
)

var (
	ErrorResourceIsLockedLocally = errors.New("Resource is locked locally.")
)

type LockInfo struct {
	GrantedCount   int
	WaitChan       chan bool
	Timestamp      int64
	GrantOnRelease []string
}

type LockManager struct {
	mutex   sync.Mutex
	clock   int64
	lockMap map[string]*LockInfo

	nodeManager *node.NodeManager
	msgHub      *comm.MessageHub
}

func (lm *LockManager) Listen(nodeManager *node.NodeManager, msgHub *comm.MessageHub) {
	lm.lockMap = make(map[string]*LockInfo, 0)

	lm.nodeManager = nodeManager
	lm.msgHub = msgHub
	lm.msgHub.Subscribe(lm,
		comm.MessageTypeRequestLock,
		comm.MessageTypeGrantLockPermission)
}

func (lm *LockManager) HandleMessage(msg *comm.Message) {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()

	fmt.Printf("Got message %s from %s\n", msg.Type, msg.SourceNode)

	switch msg.Type {
	case comm.MessageTypeRequestLock:
		var request comm.MessageRequestLock
		msg.DecodeData(&request)
		res := request.Resource
		fmt.Printf("Resource %s\n", res)
		if lm.shouldGrantPermission(request, msg.SourceNode) {
			responseMsg := comm.Message{Type: comm.MessageTypeGrantLockPermission}
			response := comm.MessageGrantLockPermission{
				Resource: res,
			}
			responseMsg.EncodeData(response)
			lm.msgHub.Send(responseMsg, msg.SourceNode)
		} else {
			lm.lockMap[res].GrantOnRelease = append(lm.lockMap[res].GrantOnRelease, msg.SourceNode)
		}
		lm.clock = request.Timestamp + 1

	case comm.MessageTypeGrantLockPermission:
		var grant comm.MessageGrantLockPermission
		msg.DecodeData(&grant)
		fmt.Printf("Resource %s\n", grant.Resource)
		lm.lockMap[grant.Resource].GrantedCount--
		if lm.lockMap[grant.Resource].GrantedCount == 0 {
			lm.lockMap[grant.Resource].WaitChan <- true
		}
	}
}

func (lm *LockManager) LockResource(resource string) error {
	msg := comm.Message{Type: comm.MessageTypeRequestLock}
	requestMsg := comm.MessageRequestLock{
		Resource:  resource,
		Timestamp: lm.clock,
	}

	if _, exists := lm.lockMap[resource]; exists {
		return ErrorResourceIsLockedLocally
	}

	msg.EncodeData(requestMsg)

	lm.mutex.Lock()
	waitChan := make(chan bool, 1)
	lm.lockMap[resource] = &LockInfo{
		WaitChan:     waitChan,
		Timestamp:    lm.clock,
		GrantedCount: len(lm.nodeManager.NodeNames()),
	}
	lm.mutex.Unlock()

	lm.msgHub.Broadcast(msg)

	<-waitChan
	return nil
}

func (lm *LockManager) UnlockResource(resource string) {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()

	msg := comm.Message{Type: comm.MessageTypeGrantLockPermission}
	grantMsg := comm.MessageGrantLockPermission{
		Resource: resource,
	}
	msg.EncodeData(grantMsg)

	for _, node := range lm.lockMap[resource].GrantOnRelease {
		lm.msgHub.Send(msg, node)
	}
	delete(lm.lockMap, resource)
}

func (lm *LockManager) shouldGrantPermission(request comm.MessageRequestLock, nodeName string) bool {
	lockInfo, exists := lm.lockMap[request.Resource]

	if !exists {
		return true
	}

	if lockInfo.GrantedCount == 0 {
		return false
	}

	if lockInfo.Timestamp > request.Timestamp {
		return false
	} else if lockInfo.Timestamp < request.Timestamp {
		return true
	} else {
		return strings.Compare(lm.nodeManager.This.Name, nodeName) < 0
	}
}
