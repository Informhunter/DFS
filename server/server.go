package server

import (
	"dfs/comm"
	c "dfs/config"
	"dfs/server/lock"
	"dfs/server/node"
	sp "dfs/server/path"
	"dfs/server/replication"
	"dfs/server/status"
	"dfs/server/token"
	"errors"
	"io"
	"mime/multipart"
	"os"
	"path"
	"sync"
)

var (
	ErrorFileAlreadyExists    = errors.New("File already exists.")
	ErrorPathIsLocked         = errors.New("Upload path is locked.")
	ErrorFileDoesNotExist     = errors.New("File does not exist.")
	ErrorFailedToRequestToken = errors.New("Failed to request token.")
)

type Server struct {
	sync.Mutex
	config             c.Config
	statusManager      status.StatusManager
	nodeManager        node.NodeManager
	tokenManager       token.TokenManager
	lockManager        lock.LockManager
	replicationManager replication.ReplicationManager
	pathManager        sp.PathManager
	msgHub             comm.MessageHub
}

func (server *Server) Start(config c.Config) {
	server.config = config

	server.nodeManager.UseConfig(&server.config)

	server.pathManager.UseConfig(&server.config)
	server.pathManager.Listen(&server.nodeManager, &server.msgHub)

	server.statusManager.UseConfig(&server.config)
	server.statusManager.Listen(&server.nodeManager, &server.msgHub)

	server.tokenManager.UseConfig(&server.config)
	server.tokenManager.Listen(
		&server.nodeManager,
		&server.statusManager,
		&server.pathManager,
		&server.msgHub)

	server.lockManager.UseConfig(&server.config)
	server.lockManager.Listen(&server.nodeManager, &server.msgHub)

	server.replicationManager.UseConfig(&server.config)
	server.replicationManager.Listen(&server.nodeManager, &server.statusManager, &server.msgHub)

	server.msgHub.Listen(&server.nodeManager, config.This.PrivateAddress)
}

func (server *Server) RequestUpload(bucketName, fileName string) (address, token string, err error) {
	server.statusManager.CountRequest()

	uploadPath := path.Join(bucketName, fileName)

	err = server.lockManager.LockResource("path:" + uploadPath)
	if err != nil {
		return "", "", err
	}
	defer server.lockManager.UnlockResource("path:" + uploadPath)

	if server.pathManager.IsLocked(uploadPath) {
		return "", "", ErrorPathIsLocked
	}

	server.pathManager.LockPath(uploadPath)

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

	newPath := path.Join(server.config.UploadDir, uploadPath)

	err = os.MkdirAll(path.Dir(newPath), 0755)
	if err != nil {
		return err
	}

	resultFile, err := os.Create(newPath)
	if err != nil {
		return err
	}

	_, err = io.Copy(resultFile, file)
	if err != nil {
		resultFile.Close()
		os.Remove(newPath)
		return err
	}

	resultFile.Close()

	server.replicationManager.ReplicateFile(uploadPath)

	return nil
}

func (server *Server) RequestDownload(bucketName, fileName string) (address, token string, err error) {
	server.statusManager.CountRequest()

	downloadPath := path.Join(bucketName, fileName)

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

	downloadPath = path.Join(server.config.UploadDir, downloadPath)

	return downloadPath, nil
}

func (server *Server) Status() map[string]status.NodeStatus {
	return server.statusManager.Status()
}
