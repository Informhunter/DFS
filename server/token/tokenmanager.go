package token

import (
	"dfs/comm"
	c "dfs/config"
	"dfs/server/node"
	"dfs/server/path"
	"dfs/server/status"
	"errors"
	"github.com/google/uuid"
	"log"
	"os"
	p "path"
	"sync"
	"time"
)

var (
	ErrorTokenDoesNotExist = errors.New("Token does not exist.")
	ErrorFileAlreadyExists = errors.New("File already exists.")
	ErrorFileDoesNotExist  = errors.New("File does not exist.")
)

type TokenInfo struct {
	ExpireTime time.Time
	Path       string
}

type TokenManager struct {
	mutex sync.Mutex

	uploadTokenMap   map[string]TokenInfo
	downloadTokenMap map[string]TokenInfo

	incomingUploadTokens   map[string]chan string
	incomingDownloadTokens map[string]chan string

	nodeManager   *node.NodeManager
	statusManager *status.StatusManager
	pathManager   *path.PathManager
	msgHub        *comm.MessageHub
	config        *c.Config
}

func (tm *TokenManager) UseConfig(config *c.Config) {
	tm.config = config
}

func (tm *TokenManager) Listen(
	nodeManager *node.NodeManager,
	statusManager *status.StatusManager,
	pathManager *path.PathManager,
	msgHub *comm.MessageHub) {

	tm.incomingUploadTokens = make(map[string]chan string, 0)
	tm.incomingDownloadTokens = make(map[string]chan string, 0)
	tm.uploadTokenMap = make(map[string]TokenInfo, 0)
	tm.downloadTokenMap = make(map[string]TokenInfo, 0)

	tm.nodeManager = nodeManager
	tm.statusManager = statusManager
	tm.pathManager = pathManager
	tm.msgHub = msgHub

	msgHub.Subscribe(tm,
		comm.MessageTypeUploadToken,
		comm.MessageTypeDownloadToken,
		comm.MessageTypeRequestUploadToken,
		comm.MessageTypeRequestDownloadToken)

	go func() {
		ticker := time.Tick(time.Minute * 1)
		for {
			<-ticker
			tm.mutex.Lock()
			now := time.Now()
			for token, tokenInfo := range tm.uploadTokenMap {
				if now.After(tokenInfo.ExpireTime) {
					tm.pathManager.UnlockPath(tokenInfo.Path)
					delete(tm.uploadTokenMap, token)
					tm.statusManager.TokenDeleted()
				}
			}
			for token, tokenInfo := range tm.downloadTokenMap {
				if now.After(tokenInfo.ExpireTime) {
					delete(tm.downloadTokenMap, token)
					tm.statusManager.TokenDeleted()
				}
			}
			tm.mutex.Unlock()
		}
	}()
}

func (tm *TokenManager) createLocalToken(path string, tokenType string) (token string, err error) {
	token = uuid.New().String()
	var tokenMap map[string]TokenInfo

	checkPath := p.Join(tm.config.UploadDir, path)

	switch tokenType {

	case "upload":
		if _, err := os.Stat(checkPath); !os.IsNotExist(err) {
			return "", ErrorFileAlreadyExists
		}
		tokenMap = tm.uploadTokenMap

	case "download":
		if _, err := os.Stat(checkPath); os.IsNotExist(err) {
			return "", ErrorFileDoesNotExist
		}
		tokenMap = tm.downloadTokenMap
	default:
		log.Fatal("Bad")
	}

	tokenMap[token] = TokenInfo{
		Path:       path,
		ExpireTime: time.Now().Add(time.Minute * 2),
	}

	tm.statusManager.TokenAdded()
	return token, nil
}

func (tm *TokenManager) GetPathByToken(token string, tokenType string) (path string, err error) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	var tokenMap map[string]TokenInfo

	switch tokenType {
	case "upload":
		tokenMap = tm.uploadTokenMap
	case "download":
		tokenMap = tm.downloadTokenMap
	default:
		log.Fatal("Bad")
	}

	tokenInfo, exists := tokenMap[token]
	if !exists {
		return "", ErrorTokenDoesNotExist
	}
	delete(tokenMap, token)
	tm.statusManager.TokenDeleted()
	return tokenInfo.Path, nil
}

func (tm *TokenManager) RequestToken(path string, nodeName string, tokenType string) (token string) {

	if nodeName == tm.nodeManager.This.Name {
		tm.mutex.Lock()
		defer tm.mutex.Unlock()
		token, err := tm.createLocalToken(path, tokenType)
		if err != nil {
			return ""
		}
		return token
	}

	var requestMsg comm.Message

	switch tokenType {
	case "upload":
		requestMsg.Type = comm.MessageTypeRequestUploadToken
	case "download":
		requestMsg.Type = comm.MessageTypeRequestDownloadToken
	default:
		log.Fatal("Bad")
	}

	request := comm.MessageRequestToken{
		Path: path,
	}
	requestMsg.EncodeData(request)

	answerChan := make(chan string, 1)

	tm.mutex.Lock()

	switch tokenType {
	case "upload":
		tm.incomingUploadTokens[path] = answerChan
	case "download":
		tm.incomingDownloadTokens[path] = answerChan
	default:
		log.Fatal("Bad")
	}

	tm.mutex.Unlock()

	tm.msgHub.Send(requestMsg, nodeName)
	token = <-answerChan

	return token
}

func (tm *TokenManager) HandleMessage(msg *comm.Message) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	switch msg.Type {

	case comm.MessageTypeRequestUploadToken:
		var request comm.MessageRequestToken

		err := msg.DecodeData(&request)
		if err != nil {
			return
		}

		token, err := tm.createLocalToken(request.Path, "upload")
		if err != nil {
			token = ""
		}

		responseMsg := comm.Message{Type: comm.MessageTypeUploadToken}
		tokenMessage := comm.MessageToken{
			Path:  request.Path,
			Token: token,
		}
		responseMsg.EncodeData(tokenMessage)
		tm.msgHub.Send(responseMsg, msg.SourceNode)

	case comm.MessageTypeUploadToken:
		var response comm.MessageToken

		err := msg.DecodeData(&response)
		if err != nil {
			return
		}

		channel, exists := tm.incomingUploadTokens[response.Path]
		if !exists {
			return
		}

		channel <- response.Token

	case comm.MessageTypeRequestDownloadToken:
		var request comm.MessageRequestToken
		msg.DecodeData(&request)

		token, err := tm.createLocalToken(request.Path, "download")
		if err != nil {
			token = ""
		}

		responseMsg := comm.Message{Type: comm.MessageTypeDownloadToken}
		tokenMessage := comm.MessageToken{
			Path:  request.Path,
			Token: token,
		}
		responseMsg.EncodeData(tokenMessage)
		tm.msgHub.Send(responseMsg, msg.SourceNode)

	case comm.MessageTypeDownloadToken:
		var response comm.MessageToken

		err := msg.DecodeData(&response)
		if err != nil {
			return
		}

		channel, exists := tm.incomingDownloadTokens[response.Path]
		if !exists {
			return
		}

		channel <- response.Token
	}
}
