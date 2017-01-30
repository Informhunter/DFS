package replication

import (
	"dfs/comm"
	c "dfs/config"
	"dfs/server/node"
	"dfs/server/status"
	"os"
	p "path"
)

type ReplicationManager struct {
	config        *c.Config
	nodeManager   *node.NodeManager
	statusManager *status.StatusManager
	msgHub        *comm.MessageHub
}

func (rm *ReplicationManager) UseConfig(config *c.Config) {
	rm.config = config
}

func (rm *ReplicationManager) Listen(
	nodeManager *node.NodeManager,
	statusManager *status.StatusManager,
	msgHub *comm.MessageHub) {

	rm.nodeManager = nodeManager
	rm.statusManager = statusManager
	rm.msgHub = msgHub
	rm.msgHub.Subscribe(rm, comm.MessageTypeFile, comm.MessageTypeFileReceived)
}

func (rm *ReplicationManager) ReplicateFile(path string) {
}

func (rm *ReplicationManager) HandleMessage(msg *comm.Message) {
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
	case comm.MessageTypeFileReceived:

	}
}
