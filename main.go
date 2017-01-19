// Package main provides nothing
package main

import "net/http"

func main() {
	http.HandleFunc("/request_download", requestDownload)
	http.HandleFunc("/download", download)
	http.HandleFunc("/request_upload", requestUpload)
	http.HandleFunc("/upload", upload)
	http.ListenAndServe(":80", nil)
}

func requestDownload(response http.ResponseWriter, request *http.Request) {
	response.Write([]byte("Request download"))
}

func download(response http.ResponseWriter, request *http.Request) {
	response.Write([]byte("Download"))
}

func requestUpload(response http.ResponseWriter, request *http.Request) {
	response.Write([]byte("Request upload"))
}

func upload(response http.ResponseWriter, request *http.Request) {
	response.Write([]byte("Upload"))
}
