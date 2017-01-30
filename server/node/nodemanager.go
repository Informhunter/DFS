package node

import (
	c "dfs/config"
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
	This      NodeInfo
	nodes     map[string]NodeInfo
	nodeNames []string
}

func (nm NodeManager) Node(nodeName string) NodeInfo {
	if nodeName == nm.This.Name {
		return nm.This
	}
	return nm.nodes[nodeName]
}

func (nm NodeManager) Nodes() map[string]NodeInfo {
	return nm.nodes
}

func (nm NodeManager) NodeNames() []string {
	return nm.nodeNames
}

func (nm *NodeManager) AddNode(nodeInfo NodeInfo) error {
	if nm.nodes == nil {
		nm.nodes = make(map[string]NodeInfo, 0)
	}
	if _, exists := nm.nodes[nodeInfo.Name]; exists {
		return ErrorNodeAlreadyExists
	}
	nm.nodeNames = append(nm.nodeNames, nodeInfo.Name)
	nm.nodes[nodeInfo.Name] = nodeInfo
	return nil
}

func (nm *NodeManager) UseConfig(config *c.Config) {
	nm.This.Name = config.This.Name
	nm.This.PublicAddress = config.This.PublicAddress
	nm.This.PrivateAddress = config.This.PrivateAddress

	for _, nodeInfo := range config.Nodes {
		node := NodeInfo{
			Name:           nodeInfo.Name,
			PublicAddress:  nodeInfo.PublicAddress,
			PrivateAddress: nodeInfo.PrivateAddress,
		}
		nm.AddNode(node)
	}
}
