package pkg

import (
	"net/http"
	"time"
)

var HttpClient = &http.Client{
	Timeout: 30 * time.Second,
	// Transport: &http.Transport{
	// 	MaxIdleConns:        100,
	// 	IdleConnTimeout:     90 * time.Second,
	// 	MaxIdleConnsPerHost: 20,
	// },
	Transport: http.DefaultTransport,
}
