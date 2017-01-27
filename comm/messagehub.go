package comm

import (
	"dfs/server/node"
	"encoding/gob"
	"net"
)

type MessageHandler interface {
	HandleMessage(*Message)
}

type MessageHub struct {
	nodeManager     *node.NodeManager
	messageHandlers map[MessageType][]MessageHandler
}

func (msgHub *MessageHub) Listen(nodeManager *node.NodeManager, addr string) error {
	msgHub.nodeManager = nodeManager
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				continue
			}
			msg := new(Message)
			dec := gob.NewDecoder(conn)
			dec.Decode(msg)
			for _, msgHandler := range msgHub.messageHandlers[msg.Type] {
				msgHandler.HandleMessage(msg)
			}
		}
	}()
	return nil
}

func (msgHub *MessageHub) Subscribe(msgHandler MessageHandler, msgTypes ...MessageType) {
	if msgHub.messageHandlers == nil {
		msgHub.messageHandlers = make(map[MessageType][]MessageHandler, 0)
	}
	for _, msgType := range msgTypes {
		msgHub.messageHandlers[msgType] = append(msgHub.messageHandlers[msgType], msgHandler)
	}
}

func (msgHub *MessageHub) Send(msg Message, nodeName string) error {
	node := msgHub.nodeManager.Node(nodeName)
	conn, err := net.Dial("tcp", node.PrivateAddress)
	if err != nil {
		return err
	}
	enc := gob.NewEncoder(conn)
	enc.Encode(msg)
	conn.Close()
	return nil
}

func (msgHub *MessageHub) Broadcast(msg Message) error {
	for _, node := range msgHub.nodeManager.Nodes() {
		conn, err := net.Dial("tcp", node.PrivateAddress)
		if err != nil {
			return err
		}
		enc := gob.NewEncoder(conn)
		enc.Encode(msg)
		conn.Close()
	}
	return nil
}
