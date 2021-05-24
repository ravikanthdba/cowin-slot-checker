package cowin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

const (
	userAgent = "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/56.0.2924.76 Safari/537.36 Mozilla/5.0 (X11; Linux x86_64) Chrome/44.0.2403.157 Thunderstorm/1.0 (Linux)"
	timeout   = 10 * time.Second
)

type Client struct {
	baseURL    *url.URL
	HTTPClient *http.Client
	userAgent  string
}

type client interface {
	NewClient(client *http.Client) (*Client, error)
	NewRequest(ctx context.Context, method string, u *url.URL, body io.Reader) (*http.Request, error)
	DoJSON(path string, request *http.Request, v interface{}) ([]byte, error)
}

func NewClient(client *http.Client, baseURL string) (*Client, error) {
	var c Client
	if client == nil {
		client = &http.Client{
			Timeout: timeout,
		}
	}

	urlValue, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("%q", err)
	}

	c.baseURL = urlValue
	c.HTTPClient = client
	c.userAgent = userAgent

	return &c, nil
}

func (c *Client) NewRequest(ctx context.Context, method string, u *url.URL, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL.ResolveReference(u).String(), body)
	if err != nil {
		return nil, fmt.Errorf("%q", "invalid http request")
	}
	req.Header.Add("User-Agent", userAgent)
	return req, nil
}

func (c *Client) DoJSON(path string, request *http.Request, v interface{}) ([]byte, error) {
	c.baseURL.Path = path
	response, err := c.HTTPClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%q", "unable to Query "+c.baseURL.String()+" app")
	}
	defer response.Body.Close()
	responseBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("%q", "unable to decode response")
	}

	if response.StatusCode != http.StatusOK {
		if response.StatusCode == http.StatusForbidden {
			return nil, fmt.Errorf("too many requests already sent to %s, please retry after 1hour", c.baseURL)
		}
		return nil, fmt.Errorf("status code is not 200, but Status Code is: %d and query is: %s", response.StatusCode, c.baseURL)
	}

	err = json.Unmarshal(responseBytes, &v)
	if err != nil {
		return nil, fmt.Errorf("%q", "unable to Unmarshal Response")
	}

	return responseBytes, nil
}
