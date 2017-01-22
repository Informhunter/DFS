// Package main provides nothing
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"text/template"
)

const (
	UploadFileKey string = "upload_file"
)

const (
	RequestDownloadURL string = "/request_download/"
	DownloadURL               = "/download/"
	RequestUploadURL          = "/request_upload/"
	UploadURL                 = "/upload/"
)

var uploadHtmlTemplate *template.Template

func main() {

	var err error
	uploadHtmlTemplate, err = template.ParseFiles("templates/upload.html")
	if err != nil {
		log.Fatal(err)
	}

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

	enc := json.NewEncoder(response)
	enc.Encode(struct {
		Address string
		Token   string
	}{
		"192.168.1.1",
		"asdfasdfasdfasdfasdf",
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

	enc := json.NewEncoder(response)
	enc.Encode(struct {
		Address string
		Token   string
	}{
		"192.168.1.1",
		"asdfasdfasdfasdfasdf",
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

	downloadToken, exists := query["download_token"]
	if !exists {
		uploadHtmlTemplate.Execute(response, struct {
			Action        string
			UploadFileKey string
		}{
			UploadURL + "?download_token=" + "some_token",
			UploadFileKey,
		})
		return
	}

	_, fileHeader, err := request.FormFile(UploadFileKey)
	if err != nil {
		uploadHtmlTemplate.Execute(response, struct {
			Action        string
			UploadFileKey string
		}{
			UploadURL + "?download_token=" + "some_token",
			UploadFileKey,
		})
		return
	}

	enc := json.NewEncoder(response)
	enc.Encode(struct {
		UploadToken string
		FileName    string
	}{
		downloadToken[0],
		fileHeader.Filename,
	})
}
