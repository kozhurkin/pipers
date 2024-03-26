package tests

import "net/http"

var httpclient http.Client

func init() {
	httpclient = http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
	}
}
