// Package main provides nothing
package main

import (
	s "dfs/server"
	"encoding/json"
	"log"
	"net/http"
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
	http.ListenAndServe(":80", nil)
}

func requestDownload(response http.ResponseWriter, request *http.Request) {
	query := request.URL.Query()
	bucketName, exists := query["bucket"]
	if !exists {
		http.NotFound(response, request)
		return
	}

	fileName, exists := query["filename"]
	if !exists {
		http.NotFound(response, request)
		return
	}

	address, token, err := server.RequestDownload(bucketName[0], fileName[0])
	if err != nil {
		http.Error(response, err.Error(), 403)
		return
	}

	downloadHtmlTemplate.Execute(response, struct {
		DownloadURL      string
		DownloadFileName string
	}{
		DownloadURL + "?download_token=" + token,
		fileName[0],
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
		bucketName[0],
		fileName[0],
	})
}

func download(response http.ResponseWriter, request *http.Request) {
	query := request.URL.Query()
	downloadToken, exists := query["download_token"]
	if !exists {
		http.NotFound(response, request)
		return
	}

	downloadPath, err := server.Download(downloadToken[0])
	if err != nil {
		http.Error(response, err.Error(), 403)
		return
	}

	http.ServeFile(response, request, downloadPath)

	enc := json.NewEncoder(response)
	enc.Encode(struct {
		DownloadToken string
	}{
		downloadToken[0],
	})
}

func requestUpload(response http.ResponseWriter, request *http.Request) {
	query := request.URL.Query()
	bucketName, exists := query["bucket"]
	if !exists {
		http.NotFound(response, request)
		return
	}

	fileName, exists := query["filename"]
	if !exists {
		http.NotFound(response, request)
		return
	}

	address, token, err := server.RequestUpload(bucketName[0], fileName[0])
	if err != nil {
		http.Error(response, err.Error(), 403)
		return
	}

	uploadHtmlTemplate.Execute(response, struct {
		Action        string
		UploadFileKey string
	}{
		UploadURL + "?upload_token=" + token,
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
		bucketName[0],
		fileName[0],
	})
}

func upload(response http.ResponseWriter, request *http.Request) {
	query := request.URL.Query()

	uploadToken, exists := query["upload_token"]
	if !exists {
		uploadHtmlTemplate.Execute(response, struct {
			Action        string
			UploadFileKey string
		}{
			UploadURL + "?upload_token=" + "some_token",
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
			UploadURL + "?upload_token=" + "some_token",
			UploadFileKey,
		})
		return
	}

	err = server.Upload(uploadToken[0], file, fileHeader)
	if err != nil {
		http.NotFound(response, request)
		return
	}

	enc := json.NewEncoder(response)
	enc.Encode(struct {
		UploadToken string
		FileName    string
	}{
		uploadToken[0],
		fileHeader.Filename,
	})
}
