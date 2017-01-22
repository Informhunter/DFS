package server

import (
	"errors"
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
	tokenMap    map[string]string
	lockedPaths map[string]bool
}

func (server *Server) Start() {
	server.tokenMap = make(map[string]string, 0)
	server.lockedPaths = make(map[string]bool, 0)
}

func (server *Server) RequestUpload(bucketName, fileName string) (address, token string, err error) {
	token = uuid.New().String()
	uploadPath := path.Join(UploadDir, bucketName, fileName)

	server.Lock()
	defer server.Unlock()

	if _, err := os.Stat(uploadPath); !os.IsNotExist(err) {
		return "", "", ErrorFileAlreadyExists
	}

	if _, exists := server.lockedPaths[uploadPath]; exists {
		return "", "", ErrorPathIsLocked
	}

	server.tokenMap[token] = uploadPath
	server.lockedPaths[uploadPath] = true

	return "some upload address", token, nil
}

func (server *Server) Upload(token string, file multipart.File, fileHeader *multipart.FileHeader) (err error) {
	server.Lock()
	defer server.Unlock()
	uploadPath, exists := server.tokenMap[token]
	if !exists {
		return ErrorTokenDoesNotExist
	}
	delete(server.tokenMap, token)
	delete(server.lockedPaths, uploadPath)

	err = os.MkdirAll(path.Dir(uploadPath), 0777)
	if err != nil {
		return err
	}

	resultFile, err := os.Create(uploadPath)
	if err != nil {
		return err
	}

	defer resultFile.Close()

	io.Copy(resultFile, file)
	return nil
}

func (server *Server) RequestDownload(bucketName, fileName string) (address, token string, err error) {
	token = uuid.New().String()
	downloadPath := path.Join(UploadDir, bucketName, fileName)

	server.Lock()
	defer server.Unlock()

	if _, err := os.Stat(downloadPath); os.IsNotExist(err) {
		return "", "", ErrorFileDoesNotExist
	}

	if _, exists := server.lockedPaths[downloadPath]; exists {
		return "", "", ErrorPathIsLocked
	}

	server.tokenMap[token] = downloadPath
	return "some download address", token, nil
}
func (server *Server) Download(token string) (downloadPath string, err error) {
	server.Lock()
	defer server.Unlock()

	downloadPath, exists := server.tokenMap[token]

	if !exists {
		return "", ErrorTokenDoesNotExist
	}
	delete(server.tokenMap, token)

	return downloadPath, err
}
