package token

import (
	"dfs/comm"
	"dfs/server/node"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"os"
	"sync"
	"time"
)

var (
	ErrorTokenDoesNotExist = errors.New("Token does not exist.")
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

	nodeManager *node.NodeManager
	msgHub      *comm.MessageHub
}

func (tm *TokenManager) Listen(nodeManager *node.NodeManager, msgHub *comm.MessageHub) {
	tm.incomingUploadTokens = make(map[string]chan string, 0)
	tm.incomingDownloadTokens = make(map[string]chan string, 0)
	tm.uploadTokenMap = make(map[string]TokenInfo, 0)
	tm.downloadTokenMap = make(map[string]TokenInfo, 0)

	tm.nodeManager = nodeManager
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
					delete(tm.uploadTokenMap, token)
				}
			}
			for token, tokenInfo := range tm.downloadTokenMap {
				if now.After(tokenInfo.ExpireTime) {
					delete(tm.uploadTokenMap, token)
				}
			}
			tm.mutex.Unlock()
		}
	}()
}

func (tm *TokenManager) GetUploadPath(token string) (path string, err error) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	tokenInfo, exists := tm.uploadTokenMap[token]
	if !exists {
		return "", ErrorTokenDoesNotExist
	}
	delete(tm.uploadTokenMap, token)
	return tokenInfo.Path, nil
}

func (tm *TokenManager) RequestUploadToken(path string, nodeName string) (token string) {
	requestMsg := comm.Message{Type: comm.MessageTypeRequestUploadToken}
	request := comm.MessageRequestToken{
		Path: path,
	}
	requestMsg.EncodeData(request)

	answerChan := make(chan string, 1)

	tm.mutex.Lock()
	tm.incomingUploadTokens[path] = answerChan
	tm.mutex.Unlock()

	tm.msgHub.Send(requestMsg, nodeName)
	token = <-answerChan

	return token
}

func (tm *TokenManager) RequestDownloadToken(path string, nodeName string) (token string) {
	requestMsg := comm.Message{Type: comm.MessageTypeRequestDownloadToken}
	request := comm.MessageRequestToken{
		Path: path,
	}
	requestMsg.EncodeData(request)

	answerChan := make(chan string, 1)

	tm.mutex.Lock()
	tm.incomingDownloadTokens[path] = answerChan
	tm.mutex.Unlock()

	tm.msgHub.Send(requestMsg, nodeName)
	token = <-answerChan

	return token
}
func (tm *TokenManager) GetDownloadPath(token string) (path string, err error) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	tokenInfo, exists := tm.downloadTokenMap[token]
	if !exists {
		return "", ErrorTokenDoesNotExist
	}
	delete(tm.downloadTokenMap, token)
	return tokenInfo.Path, nil
}

func (tm *TokenManager) HandleMessage(msg *comm.Message) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	fmt.Printf("Got message from %s\n", msg.SourceNode)
	fmt.Printf("Message type: %s\n", msg.Type)

	switch msg.Type {

	case comm.MessageTypeRequestUploadToken:
		var request comm.MessageRequestToken

		err := msg.DecodeData(&request)
		if err != nil {
			return
		}

		token := uuid.New().String()
		if _, err := os.Stat(request.Path); !os.IsNotExist(err) {
			return
		}

		tm.uploadTokenMap[token] = TokenInfo{
			Path:       request.Path,
			ExpireTime: time.Now().Add(time.Minute * 2),
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
		token := uuid.New().String()
		if _, err := os.Stat(request.Path); os.IsNotExist(err) {
			return
		}

		fmt.Println("Requested path: ", request.Path)

		tm.downloadTokenMap[token] = TokenInfo{
			Path:       request.Path,
			ExpireTime: time.Now().Add(time.Minute * 2),
		}

		responseMsg := comm.Message{Type: comm.MessageTypeDownloadToken}
		tokenMessage := comm.MessageToken{
			Path:  request.Path,
			Token: token,
		}
		responseMsg.EncodeData(tokenMessage)
		tm.msgHub.Send(responseMsg, msg.SourceNode)
		fmt.Println("Response msg sent.")

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
