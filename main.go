// Package main provides nothing
package main

import (
	s "dfs/server"
	u "dfs/util"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"text/template"
)

const (
	UploadFileKey string = "upload_file"
)

const (
	RequestDownloadURL = "/request_download/"
	DownloadURL        = "/download/"
	RequestUploadURL   = "/request_upload/"
	UploadURL          = "/upload/"
)

var (
	ErrorBadQuery = errors.New("Bad query.")
)

var server s.Server
var uploadHtmlTemplate *template.Template
var downloadHtmlTemplate *template.Template

func main() {

	var err error

	uploadHtmlTemplate, err = template.ParseFiles("templates/upload.html")
	if err != nil {
		log.Fatal(err)
	}

	downloadHtmlTemplate, err = template.ParseFiles("templates/download.html")
	if err != nil {
		log.Fatal(err)
	}

	server.Start()

	http.HandleFunc(RequestDownloadURL, requestDownload)
	http.HandleFunc(DownloadURL, download)
	http.HandleFunc(RequestUploadURL, requestUpload)
	http.HandleFunc(UploadURL, upload)
	http.ListenAndServe("localhost:80", nil)
}

func requestDownload(response http.ResponseWriter, request *http.Request) {
	bucketName, fileName, err := extractBucketNameFileName(request)
	if err != nil {
		http.Error(response, err.Error(), 403)
		return
	}

	address, token, err := server.RequestDownload(bucketName, fileName)
	if err != nil {
		http.Error(response, err.Error(), 403)
		return
	}

	downloadHtmlTemplate.Execute(response, struct {
		DownloadURL      string
		DownloadFileName string
	}{
		DownloadURL + token,
		fileName,
	})

	enc := json.NewEncoder(response)
	enc.Encode(struct {
		Address string
		Token   string
	}{
		address,
		token,
	})

	enc.Encode(struct {
		BucketName string
		FileName   string
	}{
		bucketName,
		fileName,
	})
}

func download(response http.ResponseWriter, request *http.Request) {
	downloadToken, err := extractToken(request)
	if err != nil {
		http.Error(response, err.Error(), 403)
		return
	}

	downloadPath, err := server.Download(downloadToken)
	if err != nil {
		http.Error(response, err.Error(), 403)
		return
	}

	http.ServeFile(response, request, downloadPath)

	enc := json.NewEncoder(response)
	enc.Encode(struct {
		DownloadToken string
	}{
		downloadToken,
	})
}

func requestUpload(response http.ResponseWriter, request *http.Request) {
	bucketName, fileName, err := extractBucketNameFileName(request)
	if err != nil {
		http.Error(response, err.Error(), 403)
		return
	}

	address, token, err := server.RequestUpload(bucketName, fileName)
	if err != nil {
		http.Error(response, err.Error(), 403)
		return
	}

	uploadHtmlTemplate.Execute(response, struct {
		Action        string
		UploadFileKey string
	}{
		UploadURL + token,
		UploadFileKey,
	})

	enc := json.NewEncoder(response)
	enc.Encode(struct {
		Address string
		Token   string
	}{
		address,
		token,
	})

	enc.Encode(struct {
		BucketName string
		FileName   string
	}{
		bucketName,
		fileName,
	})
}

func upload(response http.ResponseWriter, request *http.Request) {
	uploadToken, err := extractToken(request)
	if err != nil {
		uploadHtmlTemplate.Execute(response, struct {
			Action        string
			UploadFileKey string
		}{
			UploadURL + uploadToken,
			UploadFileKey,
		})
		return
	}

	file, fileHeader, err := request.FormFile(UploadFileKey)
	if err != nil {
		uploadHtmlTemplate.Execute(response, struct {
			Action        string
			UploadFileKey string
		}{
			UploadURL + uploadToken,
			UploadFileKey,
		})
		return
	}

	err = server.Upload(uploadToken, file, fileHeader)
	if err != nil {
		http.Error(response, err.Error(), 403)
		return
	}

	enc := json.NewEncoder(response)
	enc.Encode(struct {
		UploadToken string
		FileName    string
	}{
		uploadToken,
		fileHeader.Filename,
	})
}

func extractBucketNameFileName(request *http.Request) (bucketName string, fileName string, err error) {
	parts := strings.Split(request.URL.Path[1:], "/")
	if len(parts) != 3 {
		return "", "", ErrorBadQuery
	}

	if !u.IsValidName(parts[1]) {
		return "", "", ErrorBadQuery
	}

	if !u.IsValidName(parts[2]) {
		return "", "", ErrorBadQuery
	}

	return parts[1], parts[2], nil
}

func extractToken(request *http.Request) (token string, err error) {
	parts := strings.Split(request.URL.Path[1:], "/")
	if len(parts) != 2 {
		return "", ErrorBadQuery
	}
	if !u.IsValidName(parts[1]) {
		return "", ErrorBadQuery
	}
	return parts[1], nil
}
