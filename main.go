// Package main provides nothing
package main

import (
	c "dfs/config"
	s "dfs/server"
	u "dfs/util"
	"encoding/json"
	"flag"
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
	StatusURL          = "/status/"
)

var configFileName = flag.String("config", "config.json", "Config file name")

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

	flag.Parse()

	var config c.Config
	err = config.Load(*configFileName)
	if err != nil {
		log.Fatal(err)
	}

	server.Start(config)

	http.HandleFunc(RequestDownloadURL, requestDownload)
	http.HandleFunc(DownloadURL, download)
	http.HandleFunc(RequestUploadURL, requestUpload)
	http.HandleFunc(UploadURL, upload)
	http.HandleFunc(StatusURL, status)

	http.ListenAndServe(config.This.PublicAddress, nil)
}

func requestDownload(response http.ResponseWriter, request *http.Request) {
	bucketName, fileName, err := u.ExtractBucketNameFileName(request)
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
		"http://" + address + DownloadURL + token,
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
}

func download(response http.ResponseWriter, request *http.Request) {
	downloadToken, err := u.ExtractToken(request)
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
	bucketName, fileName, err := u.ExtractBucketNameFileName(request)
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
		"http://" + address + UploadURL + token,
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
}

func upload(response http.ResponseWriter, request *http.Request) {
	uploadToken, err := u.ExtractToken(request)
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
}

func status(response http.ResponseWriter, request *http.Request) {
	enc := json.NewEncoder(response)
	enc.SetIndent("", "  ")
	status := server.Status()
	enc.Encode(status)
}
