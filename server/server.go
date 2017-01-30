package server

import (
	"dfs/comm"
	c "dfs/config"
	"dfs/server/lock"
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
	lockManager   lock.LockManager
	msgHub        comm.MessageHub
}

func (server *Server) Start(config c.Config) {
	server.nodeManager.UseConfig(config)

	server.statusManager.Listen(&server.nodeManager, &server.msgHub)
	server.tokenManager.Listen(&server.nodeManager, &server.statusManager, &server.msgHub)
	server.lockManager.Listen(&server.nodeManager, &server.msgHub)

	server.msgHub.Listen(&server.nodeManager, config.This.PrivateAddress)
}

func (server *Server) RequestUpload(bucketName, fileName string) (address, token string, err error) {
	uploadPath := path.Join(UploadDir, bucketName, fileName)

	err = server.lockManager.LockResource("path:" + uploadPath)
	if err != nil {
		return "", "", err
	}
	defer server.lockManager.UnlockResource("path:" + uploadPath)

	server.statusManager.CountRequest()

	nodeName := server.statusManager.ChooseNodeForUpload()
	token = server.tokenManager.RequestToken(uploadPath, nodeName, "upload")

	if token == "" {
		return "", "", ErrorFailedToRequestToken
	}

	return server.nodeManager.Node(nodeName).PublicAddress, token, nil
}

func (server *Server) Upload(token string, file multipart.File, fileHeader *multipart.FileHeader) (err error) {
	server.statusManager.CountRequest()

	uploadPath, err := server.tokenManager.GetPathByToken(token, "upload")
	if err != nil {
		return err
	}

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
	server.statusManager.CountRequest()

	downloadPath := path.Join(UploadDir, bucketName, fileName)

	nodeName := server.statusManager.ChooseNodeForDownload()
	token = server.tokenManager.RequestToken(downloadPath, nodeName, "download")
	if token == "" {
		return "", "", ErrorFailedToRequestToken
	}

	return server.nodeManager.Node(nodeName).PublicAddress, token, nil
}

func (server *Server) Download(token string) (downloadPath string, err error) {
	server.statusManager.CountRequest()

	downloadPath, err = server.tokenManager.GetPathByToken(token, "download")
	if err != nil {
		return "", err
	}

	return downloadPath, err
}

func (server *Server) Status() map[string]status.NodeStatus {
	return server.statusManager.Status()
}
