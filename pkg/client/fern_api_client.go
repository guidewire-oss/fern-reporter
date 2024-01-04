package client

import (
	"net/http"
	"time"
)

type FernApiClient struct {
	name       string
	httpClient *http.Client
	baseURL    string
}

type ClientOption func(*FernApiClient)

func New(testName string, options ...ClientOption) *FernApiClient {
	f := &FernApiClient{
		name:       testName,
		httpClient: http.DefaultClient,
	}

	for _, o := range options {
		o(f)
	}

	return f
}

func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(ac *FernApiClient) {
		ac.httpClient = httpClient
	}
}

func WithBaseURL(baseURL string) ClientOption {
	return func(ac *FernApiClient) {
		ac.baseURL = baseURL
	}
}

func WithTimeout(timeout time.Duration) ClientOption {
	return func(ac *FernApiClient) {
		ac.httpClient.Timeout = timeout
	}
}
