package cowin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"io"
	"context"
	"time"
)

const (
	cowinAPI = "https://cdn-api.co-vin.in/"
)

type CowinClient struct {
	client *Client
}

func NewCowinClient() (CowinClient, error) {
	var cowinclient CowinClient
	client, err := NewClient(nil, cowinAPI)
	if err != nil {
		return cowinclient, err
	}
	cowinclient.client = client
	return cowinclient, nil
}

func (c *CowinClient) addQueryParameters(districtID string, date string) string {
	var query = make(url.Values)
	query.Set("district_id", districtID)
	query.Set("date", date)
	return query.Encode()
}

func (c *CowinClient) NewRequest(method string, u *url.URL, body io.Reader) (*http.Request, error) {
	req, err := c.client.NewRequest(context.Background(), method, u, body)
	if err != nil {
		return nil, err
	}

	return req, nil
}

func queryCowin(c *CowinClient, request *http.Request, path string, v interface{}) ([]byte, error) {
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

func (c *CowinClient) GetStates(request *http.Request) (CowinStatesResponse, error) {
	var cowinstateresponse CowinStatesResponse
	response, err := queryCowin(c, request, "/api/v2/admin/location/states", cowinstateresponse)
	if err != nil {
		return cowinstateresponse, fmt.Errorf("Error: %q", err)
	}
	err = json.Unmarshal(response, &cowinstateresponse)
	if err != nil {
		return cowinstateresponse, err
	}

	return cowinstateresponse, nil
}

func (c *CowinClient) GetDistricts(request *http.Request, stateID string) (CowinDistrictsResponse, error) {
	var cowindistrictresponse CowinDistrictsResponse
	response, err := queryCowin(c, request, "/api/v2/admin/location/districts/" + stateID, cowindistrictresponse)
	if err != nil {
		return cowindistrictresponse, err
	}
	err = json.Unmarshal(response, &cowindistrictresponse)
	if err != nil {
		return cowindistrictresponse, err
	}

	return cowindistrictresponse, nil
}

func (c *CowinClient) GetHospitals(request *http.Request, districtID string) (CowinHospitalsResponse, error) {
	var cowinhospitalsresponse CowinHospitalsResponse
	date := time.Now().Format("02-01-2006")
	c.client.baseURL.RawQuery = c.addQueryParameters(districtID, date)
	response, err := queryCowin(c, request, "/api/v2/appointment/sessions/public/calendarByDistrict", cowinhospitalsresponse)
	if err != nil {
		return cowinhospitalsresponse, err
	}

	err = json.Unmarshal(response, &cowinhospitalsresponse)
	if err != nil {
		return cowinhospitalsresponse, err
	}

	return cowinhospitalsresponse, nil
}
