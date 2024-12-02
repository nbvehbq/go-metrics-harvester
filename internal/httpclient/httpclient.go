package httpclient

import "net/http"

type HTTPClient struct {
	http.Client
}

func (h *HTTPClient) Do(req *http.Request) (*http.Response, error) {
	return h.Client.Do(req)
}
