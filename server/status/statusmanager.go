// Package statusmanager provides
package status

import (
	"dfs/comm"
	"dfs/server/node"
	//"fmt"
	"math/rand"
	"sync"
	"time"
)

type NodeStatus struct {
	RequestsPerMinute int
	RequestCounter    int
	TokenCount        int
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
	sm.nodeStatuses = make(map[string]NodeStatus, 0)
	sm.msgHub = msgHub
	sm.msgHub.Subscribe(sm, comm.MessageTypeStatus)
	go func() {
		ticker := time.Tick(time.Second * 10)
		for {
			<-ticker

			sm.mutex.Lock()

			sm.this.RequestsPerMinute = sm.this.RequestCounter * 6
			sm.this.RequestCounter = 0
			sm.nodeStatuses[sm.nodeManager.This.Name] = sm.this

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
	}()
}

func (sm *StatusManager) HandleMessage(msg *comm.Message) {
	switch msg.Type {
	case comm.MessageTypeStatus:
		var status comm.MessageNodeStatus
		err := msg.DecodeData(&status)
		if err != nil {
			return
		}

		//fmt.Printf("Got message from %s\n%s\n", msg.SourceNode, status.String())
		sm.nodeStatuses[msg.SourceNode] = NodeStatus{
			status.RequestsPerMinute,
			status.RequestCounter,
			status.TokenCount,
		}
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

func (sm StatusManager) Status() map[string]NodeStatus {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	return sm.nodeStatuses
}

func (sm StatusManager) ChooseNodeForUpload() (nodeName string) {
	nodeNames := sm.nodeManager.NodeNames()
	index := rand.Int() % len(nodeNames)
	return nodeNames[index]
}

func (sm StatusManager) ChooseNodeForDownload() (nodeName string) {
	nodeNames := sm.nodeManager.NodeNames()
	index := rand.Int() % len(nodeNames)
	return nodeNames[index]
}
