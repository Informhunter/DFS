package server

import (
	"dfs/comm"
	c "dfs/config"
	"dfs/server/node"
	"dfs/server/status"
	"dfs/server/token"
	"errors"
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
	ErrorFileAlreadyExists    = errors.New("File already exists.")
	ErrorPathIsLocked         = errors.New("Upload path is locked.")
	ErrorFileDoesNotExist     = errors.New("File does not exist.")
	ErrorFailedToRequestToken = errors.New("Failed to request token.")
)

type Server struct {
	sync.Mutex
	statusManager status.StatusManager
	nodeManager   node.NodeManager
	tokenManager  token.TokenManager
	msgHub        comm.MessageHub
	tokenMap      map[string]string
}

func (server *Server) Start(config c.Config) {
	server.tokenMap = make(map[string]string, 0)

	server.nodeManager.UseConfig(config)

	server.statusManager.Listen(&server.nodeManager, &server.msgHub)
	server.tokenManager.Listen(&server.nodeManager, &server.msgHub)

	server.msgHub.Listen(&server.nodeManager, config.This.PrivateAddress)
}

func (server *Server) RequestUpload(bucketName, fileName string) (address, token string, err error) {
	uploadPath := path.Join(UploadDir, bucketName, fileName)

	server.Lock()
	defer server.Unlock()

	server.statusManager.CountRequest()

	nodeName := server.statusManager.ChooseNodeForUpload()
	token = server.tokenManager.RequestUploadToken(uploadPath, nodeName)

	if token == "" {
		return "", "", ErrorFailedToRequestToken
	}

	return server.nodeManager.Node(nodeName).PublicAddress, token, nil
}

func (server *Server) Upload(token string, file multipart.File, fileHeader *multipart.FileHeader) (err error) {
	server.Lock()
	defer server.Unlock()

	server.statusManager.CountRequest()

	uploadPath, err := server.tokenManager.GetUploadPath(token)
	if err != nil {
		return err
	}

	server.statusManager.TokenDeleted()

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
	downloadPath := path.Join(UploadDir, bucketName, fileName)

	server.Lock()
	defer server.Unlock()

	server.statusManager.CountRequest()

	nodeName := server.statusManager.ChooseNodeForDownload()
	token = server.tokenManager.RequestDownloadToken(downloadPath, nodeName)
	if token == "" {
		return "", "", ErrorFailedToRequestToken
	}

	return server.nodeManager.Node(nodeName).PublicAddress, token, nil
}

func (server *Server) Download(token string) (downloadPath string, err error) {
	server.Lock()
	defer server.Unlock()

	server.statusManager.CountRequest()

	downloadPath, err = server.tokenManager.GetDownloadPath(token)
	if err != nil {
		return "", err
	}

	server.statusManager.TokenDeleted()

	return downloadPath, err
}

func (server *Server) Status() map[string]status.NodeStatus {
	return server.statusManager.Status()
}
