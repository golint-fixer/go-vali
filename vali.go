// Package vali provides a client interface to CIVL's Open Validation Server.
// See http://vali.fai-civl.org/webservice.html.
package vali

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"

	"golang.org/x/net/context"
	"golang.org/x/net/context/ctxhttp"
)

var endpoint = "http://vali.fai-civl.org/api/vali/json"

// A Response represents a response from the server.
type Response struct {
	Result string `json:"result"`
	Status string `json:"status"`
	Msg    string `json:"msg"`
	IGC    string `json:"igc"`
	Ref    string `json:"ref"`
	Server string `json:"server"`
}

func (r *Response) Error() string {
	return fmt.Sprintf("vali: %s", r.Msg)
}

// A ServerError represents a server error.
type ServerError struct {
	HTTPStatusCode int
	HTTPStatus     string
}

func (se *ServerError) Error() string {
	return fmt.Sprintf("vali: %d %s", se.HTTPStatusCode, se.HTTPStatus)
}

// An Option is an option for configuring a Service.
type Option func(*Service)

// Client sets the http.Client.
func Client(client *http.Client) Option {
	return func(s *Service) {
		s.client = client
	}
}

// Endpoint sets the HTTP endpoint.
func Endpoint(endpoint string) Option {
	return func(s *Service) {
		s.endpoint = endpoint
	}
}

// A Service represents a validator service.
type Service struct {
	client   *http.Client
	endpoint string
}

// NewService returns a new Service.
func NewService(options ...Option) *Service {
	s := &Service{
		client:   &http.Client{},
		endpoint: endpoint,
	}
	for _, o := range options {
		o(s)
	}
	return s
}

// ValidateIGC validates igcFile.
func (s *Service) ValidateIGC(ctx context.Context, filename string, igcFile io.Reader) error {
	b := &bytes.Buffer{}
	w := multipart.NewWriter(b)
	fw, err := w.CreateFormFile("igcfile", filename)
	if err != nil {
		return err
	}
	if _, err = io.Copy(fw, igcFile); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	req, err := http.NewRequest("POST", s.endpoint, b)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := ctxhttp.Do(ctx, s.client, req)
	if err != nil {
		return err
	}
	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if err := resp.Body.Close(); err != nil {
		return err
	}
	if resp.StatusCode < 200 || 300 <= resp.StatusCode {
		return &ServerError{
			HTTPStatusCode: resp.StatusCode,
			HTTPStatus:     resp.Status,
		}
	}
	var r Response
	if err := json.Unmarshal(body, &r); err != nil {
		return err
	}
	if r.Result != "PASSED" {
		return &r
	}
	return nil
}
