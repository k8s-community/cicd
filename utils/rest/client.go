package rest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// REST client

// Client manages communication with the API.
type Client struct {
	// HTTP client used to communicate with the API.
	client *http.Client

	// Base URL for API requests.
	baseURL *url.URL
}

// NewClient returns a new API client.
// If a nil httpClient is provided, http.DefaultClient will be used.
func NewClient(httpClient *http.Client, baseURLS string) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	urlU, _ := url.Parse(baseURLS)

	c := &Client{
		client:  httpClient,
		baseURL: urlU,
	}

	return c
}

// NewRequest creates an API request. A relative URL can be provided in urlStr,
// in which case it is resolved relative to the BaseURL of the Client.
// Relative URLs should always be specified without a preceding slash.  If
// specified, the value pointed to by body is JSON encoded and included as the
// request body.
func (c *Client) NewRequest(method, urlStr string, body interface{}) (*http.Request, error) {
	rel, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	if c.baseURL == nil {
		return nil, errors.New("Base URL is not set")
	}
	u := c.baseURL.ResolveReference(rel)

	var buf io.ReadWriter
	if body != nil {
		buf = new(bytes.Buffer)
		err := json.NewEncoder(buf).Encode(body)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// Response is an API response.
// This wraps the standard http.Response.
type Response struct {
	*http.Response
}

// newResponse creates a new Response for the provided http.Response.
func newResponse(r *http.Response) *Response {
	response := &Response{Response: r}
	return response
}

// Do sends an API request and returns the API response.  The API response is
// JSON decoded and stored in the value pointed to by v, or returned as an
// error if an API error has occurred.  If v implements the io.Writer
// interface, the raw response body will be written to v, without attempting to
// first decode it.
func (c *Client) Do(req *http.Request, v interface{}) (*Response, error) {
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	if v != nil {
		defer func() {
			err := resp.Body.Close()
			if err != nil {
				fmt.Printf("Error when closing response body: %s\n", err)
			}
		}()

		err = json.NewDecoder(resp.Body).Decode(v)
		if err == io.EOF {
			err = nil // ignore EOF errors caused by empty response body
		}
		if err != nil {
			return nil, err
		}
	}

	response := newResponse(resp)
	return response, err
}
