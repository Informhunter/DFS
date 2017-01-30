package lock

import (
	"dfs/comm"
	"dfs/server/node"
	"strings"
	"sync"
)

var (
	ErrorResourceIsLockedLocally = "Resource is locked locally."
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
	lockMap map[string]LockInfo

	nodeManager *node.NodeManager
	msgHub      *comm.MessageHub
}

func (lm *LockManager) Listen(nodeManager *node.NodeManager, msgHub *comm.MessageHub) {
	lm.nodeManager = nodeManager
	lm.msgHub = msgHub
	lm.awaitingPermission = make(map[string]LockInfo, 0)
}

func (lm *LockManager) HandleMessage(msg comm.Message) {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()
	switch msg.Type {

	case comm.MessageTypeRequestLock:
		var request comm.MessageRequestLock
		msg.DecodeData(&request)
		res := request.Resource
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

	case comm.MessageTypeGrantLockPermission:
		var grant comm.MessageGrantLockPermission
		msg.DecodeData(&grant)
		lm.lockMap[grant.Resource].GrantedCount--
		if lm.lockMap[grant.Resource].GrantedCount == 0 {
			lm.lockMap[grant.Resource].WaitChan <- true
		}
	}
}

func (lm *LockManager) LockResource(resource string) error {
	msg := comm.Message{Type: comm.MessageTypeRequestLock}
	requestMsg := comm.MessageRequestLock{
		Resource: resource,
		Clock:    lm.clock,
	}

	if _, exists := lm.lockMap[resource]; exists {
		return ErrorResourceIsLockedLocally
	}

	msg.EncodeData(requestMsg)

	lm.mutex.Lock()
	waitChan := make(chan bool, 1)
	lm.lockMap[resource].WaitChan = waitChan
	lm.lockMap[resource].Timestamp = lm.clock
	lm.lockMap[resource].GrantedCount = len(lm.nodeManager.NodeNames())
	lm.mutex.Unlock()

	lm.msgHub.Broadcast(msg)

	<-waitChan
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
		msgHub.Send(msg, node)
	}
	delete(lm.lockMap[resource], resource)
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
