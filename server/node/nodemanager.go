package node

import (
	"errors"
)

var (
	ErrorNodeAlreadyExists = errors.New("Node with this name already exists.")
)

type NodeInfo struct {
	Name           string
	PublicAddress  string
	PrivateAddress string
}

type NodeManager struct {
	This  NodeInfo
	nodes map[string]NodeInfo
}

func (nm NodeManager) Node(nodeName string) NodeInfo {
	return nm.nodes[nodeName]
}

func (nm NodeManager) Nodes() map[string]NodeInfo {
	return nm.nodes
}

func (nm *NodeManager) AddNode(nodeInfo NodeInfo) error {
	if nm.nodes == nil {
		nm.nodes = make(map[string]NodeInfo, 0)
	}
	if _, exists := nm.nodes[nodeInfo.Name]; exists {
		return ErrorNodeAlreadyExists
	}
	nm.nodes[nodeInfo.Name] = nodeInfo
	return nil
}
