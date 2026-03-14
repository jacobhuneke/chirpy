package auth

import (
	"errors"
	"net/http"
	"strings"
)

func GetAPIKey(headers http.Header) (string, error) {
	str := headers.Get("Authorization")
	if str == "" {
		return "", errors.New("unable to get apikey")
	}
	spl := strings.Split(str, " ")
	if len(spl) != 2 || spl[0] != "ApiKey" {
		return "", errors.New("unable to get apikey")
	}
	return spl[1], nil
}
