//Package comm provides message-based communication between several nodes
package comm

import (
	"dfs/server/node"
	"encoding/gob"
	"net"
)

//MessageHandler is the interface that must be implemented to be able to subscribe to incoming messages
type MessageHandler interface {
	HandleMessage(*Message)
}

//MessageHub handles all the netwoking between nodes.
//It provides methods to send messages, broadcast messages, subscribe to recive certain message types
type MessageHub struct {
	nodeManager     *node.NodeManager
	messageHandlers map[MessageType][]MessageHandler
	outConnMap      map[string]net.Conn
}

//Listen method starts listening for incoming connections and messages from other nodes
func (msgHub *MessageHub) Listen(nodeManager *node.NodeManager, addr string) error {
	msgHub.outConnMap = make(map[string]net.Conn, 0)
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
			go func() {
				for {
					msg := new(Message)
					dec := gob.NewDecoder(conn)
					err := dec.Decode(msg)
					if err != nil {
						break
					}
					for _, msgHandler := range msgHub.messageHandlers[msg.Type] {
						msgHandler.HandleMessage(msg)
					}
				}
			}()
		}
	}()
	return nil
}

//Subscribe method subscribes MessageHandler to receive certain message types
func (msgHub *MessageHub) Subscribe(msgHandler MessageHandler, msgTypes ...MessageType) {
	if msgHub.messageHandlers == nil {
		msgHub.messageHandlers = make(map[MessageType][]MessageHandler, 0)
	}
	for _, msgType := range msgTypes {
		msgHub.messageHandlers[msgType] = append(msgHub.messageHandlers[msgType], msgHandler)
	}
}

//Send method sends message to other node
func (msgHub *MessageHub) Send(msg Message, nodeName string) (err error) {
	msg.SourceNode = msgHub.nodeManager.This.Name
	node := msgHub.nodeManager.Node(nodeName)
	conn, exists := msgHub.outConnMap[node.PrivateAddress]
	if !exists {
		conn, err = net.Dial("tcp", node.PrivateAddress)
		if err != nil {
			return err
		}
		msgHub.outConnMap[node.PrivateAddress] = conn
	}
	enc := gob.NewEncoder(conn)
	enc.Encode(msg)
	return nil
}

//Broadcast method broadcasts messages to all the nodes
func (msgHub *MessageHub) Broadcast(msg Message) error {
	msg.SourceNode = msgHub.nodeManager.This.Name
	for _, node := range msgHub.nodeManager.Nodes() {
		msgHub.Send(msg, node.Name)
	}
	return nil
}

//SendInNewConnection method creates new connection and uses it to send the message
func (msgHub *MessageHub) SendInNewConnection(msg Message, nodeName string) (err error) {
	msg.SourceNode = msgHub.nodeManager.This.Name
	node := msgHub.nodeManager.Node(nodeName)
	conn, err := net.Dial("tcp", node.PrivateAddress)
	if err != nil {
		return err
	}
	enc := gob.NewEncoder(conn)
	enc.Encode(msg)
	return nil
}

//BroadcastInNewConnection method creates new connections and uses them to broadcast the message
func (msgHub *MessageHub) BroadcastInNewConnection(msg Message) error {
	msg.SourceNode = msgHub.nodeManager.This.Name
	for _, node := range msgHub.nodeManager.Nodes() {
		msgHub.SendInNewConnection(msg, node.Name)
	}
	return nil
}
