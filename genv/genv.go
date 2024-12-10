package genv

import (
	"net/http"
	"net/url"
	"os"
)

func HttpProxy() func(r *http.Request) (*url.URL, error) {
	if os.Getenv("HTTPS_PROXY") != "" {
		u, err := url.Parse(os.Getenv("HTTPS_PROXY"))
		if err == nil {
			return http.ProxyURL(u)
		}
	}
	if os.Getenv("HTTP_PROXY") != "" {
		u, err := url.Parse(os.Getenv("HTTP_PROXY"))
		if err == nil {
			return http.ProxyURL(u)
		}
	}
	return nil
}
