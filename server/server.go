package server

import (
	"dfs/comm"
	"dfs/server/node"
	"dfs/server/status"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"io"
	"mime/multipart"
	"os"
	"path"
	"sync"
)

const (
	UploadDir = "uploads"
)

var (
	ErrorFileAlreadyExists = errors.New("File already exists.")
	ErrorPathIsLocked      = errors.New("Upload path is locked.")
	ErrorTokenDoesNotExist = errors.New("Token does not exist.")
	ErrorFileDoesNotExist  = errors.New("File does not exist.")
)

type Server struct {
	sync.Mutex
	statusManager status.StatusManager
	nodeManager   node.NodeManager
	msgHub        comm.MessageHub
	tokenMap      map[string]string
	lockedPaths   map[string]bool
}

func (server *Server) Start() {
	server.tokenMap = make(map[string]string, 0)
	server.lockedPaths = make(map[string]bool, 0)

	var addr string

	fmt.Println("My Name:")
	fmt.Scan(&server.nodeManager.This.Name)

	fmt.Println("My PrivateAddress:")
	fmt.Scan(&addr)

	server.nodeManager.This.PrivateAddress = addr
	server.nodeManager.This.PublicAddress = "localhost:80"

	var otherNode node.NodeInfo

	fmt.Println("Other node's Name:")
	fmt.Scan(&otherNode.Name)
	fmt.Println("Other node's PrivateAddress:")
	fmt.Scan(&otherNode.PrivateAddress)

	server.nodeManager.AddNode(otherNode)

	server.statusManager.Listen(&server.nodeManager, &server.msgHub)

	server.msgHub.Listen(&server.nodeManager, addr)
}

func (server *Server) RequestUpload(bucketName, fileName string) (address, token string, err error) {
	token = uuid.New().String()
	uploadPath := path.Join(UploadDir, bucketName, fileName)

	server.Lock()
	defer server.Unlock()

	server.statusManager.CountRequest()

	if _, err := os.Stat(uploadPath); !os.IsNotExist(err) {
		return "", "", ErrorFileAlreadyExists
	}

	if _, exists := server.lockedPaths[uploadPath]; exists {
		return "", "", ErrorPathIsLocked
	}

	server.tokenMap[token] = uploadPath
	server.statusManager.TokenAdded()
	server.lockedPaths[uploadPath] = true

	return "some upload address", token, nil
}

func (server *Server) Upload(token string, file multipart.File, fileHeader *multipart.FileHeader) (err error) {
	server.Lock()
	defer server.Unlock()

	server.statusManager.CountRequest()

	uploadPath, exists := server.tokenMap[token]
	if !exists {
		return ErrorTokenDoesNotExist
	}

	delete(server.tokenMap, token)
	server.statusManager.TokenDeleted()
	delete(server.lockedPaths, uploadPath)

	err = os.MkdirAll(path.Dir(uploadPath), 0755)
	if err != nil {
		return err
	}

	resultFile, err := os.Create(uploadPath)
	if err != nil {
		return err
	}

	_, err = io.Copy(resultFile, file)
	if err != nil {
		resultFile.Close()
		os.Remove(uploadPath)
		return err
	}

	resultFile.Close()

	return nil
}

func (server *Server) RequestDownload(bucketName, fileName string) (address, token string, err error) {
	token = uuid.New().String()
	downloadPath := path.Join(UploadDir, bucketName, fileName)

	server.Lock()
	defer server.Unlock()

	server.statusManager.CountRequest()

	if _, err := os.Stat(downloadPath); os.IsNotExist(err) {
		return "", "", ErrorFileDoesNotExist
	}

	if _, exists := server.lockedPaths[downloadPath]; exists {
		return "", "", ErrorPathIsLocked
	}

	server.tokenMap[token] = downloadPath
	server.statusManager.TokenAdded()

	return "some download address", token, nil
}

func (server *Server) Download(token string) (downloadPath string, err error) {
	server.Lock()
	defer server.Unlock()

	server.statusManager.CountRequest()

	downloadPath, exists := server.tokenMap[token]

	if !exists {
		return "", ErrorTokenDoesNotExist
	}
	delete(server.tokenMap, token)

	server.statusManager.TokenDeleted()

	return downloadPath, err
}

func (server *Server) Status() status.NodeStatus {
	return server.statusManager.Status()
}
