// Package statusmanager provides
package status

import (
	"dfs/comm"
	"dfs/server/node"
	"fmt"
	"sync"
	"time"
)

type NodeStatus struct {
	RequestsPerMinute int
	TokenCount        int
	RequestCounter    int
}

type StatusManager struct {
	mutex        sync.Mutex
	nodeManager  *node.NodeManager
	msgHub       *comm.MessageHub
	this         NodeStatus
	nodeStatuses map[string]NodeStatus
}

func (sm *StatusManager) Listen(nodeManager *node.NodeManager, msgHub *comm.MessageHub) {
	sm.nodeManager = nodeManager
	sm.msgHub = msgHub
	sm.msgHub.Subscribe(sm, comm.MessageTypeStatus)
	go func() {
		ticker := time.Tick(time.Second * 10)
		for {
			select {
			case <-ticker:
				sm.mutex.Lock()

				sm.this.RequestsPerMinute = sm.this.RequestCounter * 6
				sm.this.RequestCounter = 0

				status := comm.MessageNodeStatus{
					sm.this.RequestsPerMinute,
					sm.this.TokenCount,
					sm.this.RequestCounter,
				}

				sm.mutex.Unlock()

				msg := comm.Message{Type: comm.MessageTypeStatus}
				msg.EncodeData(status)
				sm.msgHub.Broadcast(msg)
			}
		}
	}()
}

func (sm *StatusManager) HandleMessage(msg *comm.Message) {
	switch msg.Type {
	case comm.MessageTypeStatus:
		var status comm.MessageNodeStatus
		msg.DecodeData(&status)
		fmt.Println("Got message\n", status.String())
	}
}

func (sm *StatusManager) CountRequest() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.this.RequestCounter += 1
}

func (sm *StatusManager) TokenAdded() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.this.TokenCount += 1
}

func (sm *StatusManager) TokenDeleted() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.this.TokenCount -= 1
}

func (sm StatusManager) Status() NodeStatus {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	return sm.this
}
