package openai

import (
	"io"
	"net/http"
	"strings"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func response(status int, body string) *http.Response {
	header := make(http.Header)
	header.Set("Content-Type", "application/json")
	return &http.Response{
		StatusCode: status,
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

type failingReadCloser struct{}

func (f failingReadCloser) Read(_ []byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}

func (f failingReadCloser) Close() error {
	return nil
}
