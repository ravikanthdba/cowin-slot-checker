package cowin

import (
	"context"
	"net/http"
	"net/url"
	"fmt"
	"encoding/json"
	"io"
)

const (
	postalCodesApi = "https://api.postalpincode.in/"
)

type PostalCodeClient struct {
	client *Client
}

func NewPostalCodeClient() (PostalCodeClient, error) {
	var postalCodeClient PostalCodeClient
	client, err := NewClient(nil, postalCodesApi)
	if err != nil {
		return postalCodeClient, err
	}

	postalCodeClient.client = client
	return postalCodeClient, nil
}

func (c *PostalCodeClient) NewRequest(method string, u *url.URL, body io.Reader) (*http.Request, error) {
	req, err := c.client.NewRequest(context.Background(), method, u, body)
	if err != nil {
		return nil, err
	}

	return req, nil
}

func queryPostalCodes(c *PostalCodeClient, request *http.Request, path string, v interface{}) ([]byte, error) {
	responseBytes, err := c.client.DoJSON(path, request, v)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(responseBytes, &v)
	if err != nil {
		return nil, err
	}

	return responseBytes, nil
}

func (c *PostalCodeClient) GetPostalCodes(request *http.Request, postalCode string) ([]PostalCodeResponse, error) {
	var postalCodeResponse []PostalCodeResponse
	response, err := queryPostalCodes(c, request, "/pincode/" + postalCode, postalCodeResponse)
	if err != nil {
		fmt.Println(err)
	}

	err = json.Unmarshal(response, &postalCodeResponse)
	if err != nil {
		return postalCodeResponse, err
	}

	return postalCodeResponse, nil
}