package jaal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"go.appointy.com/jaal/jerrors"
)

type Decoder interface {
	Unmarshal([]byte, interface{}) error
}

type defaultDecoder struct{}

func (d *defaultDecoder) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

type ClientOption func(*Client)

func WithDecoder(d Decoder) ClientOption {
	return func(c *Client) {
		c.Decoder = d
	}
}

type CallOptions struct {
	Header http.Header
}

type CallOption func(*CallOptions)

func WithHeader(h http.Header) CallOption {
	return func(o *CallOptions) {
		o.Header = h
	}
}

type Client struct {
	HttpClient *http.Client

	Url     string
	Header  http.Header
	Decoder Decoder
}

func NewHttpClient(client *http.Client, url string, header http.Header, opts ...ClientOption) *Client {
	c := &Client{
		HttpClient: client,
		Url:        url,
		Header:     header,
		Decoder:    &defaultDecoder{},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

func (c *Client) Do(query string, variables, response interface{}, opts ...CallOption) error {
	rb := struct {
		Query     string
		Variables interface{}
	}{
		Query:     query,
		Variables: variables,
	}

	var opt CallOptions
	for _, op := range opts {
		op(&opt)
	}

	hr := struct {
		Data   json.RawMessage  `json:"data"`
		Errors []*jerrors.Error `json:"errors"`
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
		return fmt.Errorf("jaal: unable to decode response into graphql std format: %v", err)
	}

	if len(hr.Errors) > 0 {
		return &jerrors.MultiError{Errors: hr.Errors}
	}

	return c.Decoder.Unmarshal(hr.Data, response)
}
