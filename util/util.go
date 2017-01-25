package util

import (
	"errors"
	"net/http"
	"strings"
)

var (
	ErrorBadQuery = errors.New("Bad query.")
)

func ExtractBucketNameFileName(request *http.Request) (bucketName string, fileName string, err error) {
	parts := strings.Split(request.URL.Path[1:], "/")
	if len(parts) != 3 {
		return "", "", ErrorBadQuery
	}

	if !IsValidName(parts[1]) {
		return "", "", ErrorBadQuery
	}

	if !IsValidName(parts[2]) {
		return "", "", ErrorBadQuery
	}

	return parts[1], parts[2], nil
}

func ExtractToken(request *http.Request) (token string, err error) {
	parts := strings.Split(request.URL.Path[1:], "/")
	if len(parts) != 2 {
		return "", ErrorBadQuery
	}
	if !IsValidName(parts[1]) {
		return "", ErrorBadQuery
	}
	return parts[1], nil
}

func IsValidName(s string) bool {
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '.' && r != '-' {
			return false
		}
	}
	return true
}
