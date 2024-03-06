package gitlab_ci

import "net/http"

type authenticatedHttpTransport struct {
	authHeader string
}

func (t *authenticatedHttpTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("Authorization", t.authHeader)
	return http.DefaultTransport.RoundTrip(req)
}
