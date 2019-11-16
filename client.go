package jaal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"go.appointy.com/jaal/internal"
)

type ClientOptions struct {
	Header http.Header
}

type ClientOption func(*ClientOptions)

type Decoder interface {
	Unmarshal([]byte, interface{}) error
}

func WithHeader(h http.Header) ClientOption {
	return func(o *ClientOptions) {
		o.Header = h
	}
}

type Client struct {
	HttpClient *http.Client

	Url     string
	Header  http.Header
	Decoder Decoder
}

func NewHttpClient(client *http.Client, url string, header http.Header, decoder Decoder) *Client {
	return &Client{
		HttpClient: client,
		Url:        url,
		Header:     header,
		Decoder:    decoder,
	}
}

func (c *Client) Do(query string, variables, response interface{}, opts ...ClientOption) error {
	rb := struct {
		Query     string
		Variables interface{}
	}{
		Query:     query,
		Variables: variables,
	}

	var opt ClientOptions
	for _, op := range opts {
		op(&opt)
	}

	hr := struct {
		Data   json.RawMessage   `json:"data"`
		Errors []*internal.Error `json:"errors"`
	}{}

	data, err := json.Marshal(&rb)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.Url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("jaal: this is a bug in the library please report: %v", err)
	}
	defer req.Body.Close()

	for k, v := range c.Header {
		req.Header[k] = v
	}
	for k, v := range opt.Header {
		req.Header[k] = v
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	res, err := c.HttpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode%200 >= 100 {
		return fmt.Errorf("jaal: returned a non-success status code: %d", res.StatusCode)
	}

	if err := json.NewDecoder(res.Body).Decode(&hr); err != nil {
		return fmt.Errorf("jaal: unable to decode response into graphql std format: %w", err)
	}

	if len(hr.Errors) > 0 {
		return &MultiError{Errors: hr.Errors}
	}

	if c.Decoder != nil {
		return c.Decoder.Unmarshal(hr.Data, response)
	}

	return json.Unmarshal(hr.Data, response)
}

type MultiError struct {
	Errors []*internal.Error
}

func (e *MultiError) Error() string {
	var s strings.Builder

	for _, e := range e.Errors {
		s.WriteString(e.Error())
		s.WriteString("\n")
	}

	return s.String()
}
