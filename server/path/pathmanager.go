package path

import (
	"dfs/comm"
	c "dfs/config"
	"dfs/server/node"
	"dfs/server/status"
	"errors"
	"sync"
)

var (
	ErrorPathIsLocked = errors.New("Path is locked.")
)

type lockInfo struct {
	WaitChan chan bool
	Counter  int
}

type PathManager struct {
	mutex         sync.Mutex
	config        *c.Config
	nodeManager   *node.NodeManager
	statusManager *status.StatusManager
	msgHub        *comm.MessageHub

	lockedPaths map[string]*lockInfo
}

func (pm *PathManager) UseConfig(config *c.Config) {
	pm.config = config
}

func (pm *PathManager) Listen(
	nodeManager *node.NodeManager,
	msgHub *comm.MessageHub) {
	pm.lockedPaths = make(map[string]*lockInfo, 0)
	pm.nodeManager = nodeManager
	pm.msgHub = msgHub
	pm.msgHub.Subscribe(pm,
		comm.MessageTypeLockPath,
		comm.MessageTypeUnlockPath,
		comm.MessageTypePathLocked)
}

func (pm *PathManager) IsLocked(path string) bool {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	_, exists := pm.lockedPaths[path]
	return exists
}

func (pm *PathManager) LockPath(path string) error {
	if _, exists := pm.lockedPaths[path]; exists {
		return ErrorPathIsLocked
	}

	pm.mutex.Lock()
	waitChan := make(chan bool, 1)
	pm.lockedPaths[path] = &lockInfo{
		WaitChan: waitChan,
		Counter:  len(pm.nodeManager.NodeNames()),
	}
	pm.mutex.Unlock()

	msg := comm.Message{Type: comm.MessageTypeLockPath}
	messageFile := comm.MessageLockPath{
		Path: path,
	}
	msg.EncodeData(messageFile)

	pm.msgHub.Broadcast(msg)

	<-waitChan
	return nil
}

func (pm *PathManager) UnlockPath(path string) {
	pm.mutex.Lock()
	delete(pm.lockedPaths, path)
	pm.mutex.Unlock()

	msg := comm.Message{Type: comm.MessageTypeUnlockPath}
	messageUnlock := comm.MessageUnlockPath{
		Path: path,
	}
	msg.EncodeData(messageUnlock)
	pm.msgHub.Broadcast(msg)
}

func (pm *PathManager) HandleMessage(msg *comm.Message) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	switch msg.Type {
	case comm.MessageTypeLockPath:
		var lockPathMessage comm.MessageLockPath
		msg.DecodeData(&lockPathMessage)

		pm.lockedPaths[lockPathMessage.Path] = nil

		responseMsg := comm.Message{Type: comm.MessageTypePathLocked}
		pathLocked := comm.MessagePathLocked{
			Path: lockPathMessage.Path,
		}
		responseMsg.EncodeData(pathLocked)
		pm.msgHub.Send(responseMsg, msg.SourceNode)

	case comm.MessageTypePathLocked:
		var pathLocked comm.MessagePathLocked
		msg.DecodeData(&pathLocked)
		pm.lockedPaths[pathLocked.Path].Counter--
		if pm.lockedPaths[pathLocked.Path].Counter == 0 {
			pm.lockedPaths[pathLocked.Path].WaitChan <- true
		}
	case comm.MessageTypeUnlockPath:
		var unlockPath comm.MessageUnlockPath
		msg.DecodeData(&unlockPath)
		delete(pm.lockedPaths, unlockPath.Path)
	}
}
