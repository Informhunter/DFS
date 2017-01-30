package replication

import (
	"dfs/comm"
	c "dfs/config"
	"dfs/server/node"
	"dfs/server/status"
	"io/ioutil"
	"os"
	p "path"
	"sync"
)

type replicationInfo struct {
	WaitChan        chan bool
	ReplicatedCount int
}

type ReplicationManager struct {
	mutex          sync.Mutex
	config         *c.Config
	nodeManager    *node.NodeManager
	statusManager  *status.StatusManager
	msgHub         *comm.MessageHub
	replicationMap map[string]*replicationInfo
}

func (rm *ReplicationManager) UseConfig(config *c.Config) {
	rm.config = config
}

func (rm *ReplicationManager) Listen(
	nodeManager *node.NodeManager,
	statusManager *status.StatusManager,
	msgHub *comm.MessageHub) {

	rm.replicationMap = make(map[string]*replicationInfo, 0)

	rm.nodeManager = nodeManager
	rm.statusManager = statusManager
	rm.msgHub = msgHub
	rm.msgHub.Subscribe(rm, comm.MessageTypeFile, comm.MessageTypeFileReceived)
}

func (rm *ReplicationManager) ReplicateFile(path string) {
	filePath := p.Join(rm.config.UploadDir, path)
	file, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer file.Close()

	fileData, err := ioutil.ReadAll(file)
	if err != nil {
		return
	}

	rm.mutex.Lock()
	waitChan := make(chan bool, 1)
	rm.replicationMap[path] = &replicationInfo{
		WaitChan:        waitChan,
		ReplicatedCount: len(rm.nodeManager.NodeNames()),
	}
	rm.mutex.Unlock()

	msg := comm.Message{Type: comm.MessageTypeFile}
	messageFile := comm.MessageFile{
		Path:     path,
		FileData: fileData,
	}
	msg.EncodeData(messageFile)

	rm.msgHub.Broadcast(msg)

	<-waitChan
}

func (rm *ReplicationManager) HandleMessage(msg *comm.Message) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	switch msg.Type {
	case comm.MessageTypeFile:
		var fileMessage comm.MessageFile
		msg.DecodeData(&fileMessage)

		uploadPath := p.Join(rm.config.UploadDir, fileMessage.Path)

		err := os.MkdirAll(p.Dir(uploadPath), 0755)
		if err != nil {
			return
		}

		resultFile, err := os.Create(uploadPath)
		if err != nil {
			return
		}

		resultFile.Write(fileMessage.FileData)
		if err != nil {
			resultFile.Close()
			os.Remove(uploadPath)
		}
		responseMsg := comm.Message{Type: comm.MessageTypeFileReceived}
		fileReceived := comm.MessageFileReceived{
			Path: fileMessage.Path,
		}
		responseMsg.EncodeData(fileReceived)
		rm.msgHub.Send(responseMsg, msg.SourceNode)

	case comm.MessageTypeFileReceived:
		var fileReceived comm.MessageFileReceived
		msg.DecodeData(&fileReceived)
		rm.replicationMap[fileReceived.Path].ReplicatedCount--
		if rm.replicationMap[fileReceived.Path].ReplicatedCount == 0 {
			rm.replicationMap[fileReceived.Path].WaitChan <- true
			delete(rm.replicationMap, fileReceived.Path)
		}
	}
}
