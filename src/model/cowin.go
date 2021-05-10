package cowin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
)

const (
	cowin     = "https://cdn-api.co-vin.in/"
	userAgent = "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/56.0.2924.76 Safari/537.36 Mozilla/5.0 (X11; Linux x86_64) Chrome/44.0.2403.157 Thunderstorm/1.0 (Linux)"
	timeout   = 10 * time.Second
)

type CowinClient struct {
	baseURL    *url.URL
	HTTPClient *http.Client
	userAgent  string
}

func NewClient(c *http.Client) (*CowinClient, error) {
	if c == nil {
		c = &http.Client{
			Timeout: timeout,
		}
	}

	var client CowinClient
	urlValue, err := url.Parse(cowin)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse URL string")
	}
	client.baseURL = urlValue
	client.HTTPClient = c
	return &client, nil
}

func (c *CowinClient) addQueryParameters(districtID string, date string) string {
	var query = make(url.Values)
	query.Set("district_id", districtID)
	query.Set("date", date)
	return query.Encode()
}

func (c *CowinClient) NewRequest(ctx context.Context, method string, u *url.URL, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL.ResolveReference(u).String(), body)
	if err != nil {
		return nil, fmt.Errorf("Invalid http Request")
	}

	req.Header.Add("User-Agent", userAgent)
	return req, nil
}

func (c *CowinClient) DoJSON(path string, request *http.Request, v interface{}) ([]byte, error) {
	c.baseURL.Path = path
	response, err := c.HTTPClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("Unable to Query Cowin app")
	}
	defer response.Body.Close()
	responseBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("Unable to decode response")
	}

	if response.StatusCode != http.StatusOK {
		if response.StatusCode == http.StatusForbidden {
			return nil, fmt.Errorf("Too many requests already sent to %s, please retry after 1hour.", c.baseURL)
		}
		return nil, fmt.Errorf("Status Code is not 200, but Status Code is: %d and query is: %s", response.StatusCode, c.baseURL)
	}

	err = json.Unmarshal(responseBytes, &v)
	if err != nil {
		return nil, fmt.Errorf("Unable to Unmarshal Response")
	}

	return responseBytes, nil
}

type CowinStatesResponse struct {
	States []struct {
		StateID   int    `json:"state_id"`
		StateName string `json:"state_name"`
	} `json:"states"`
}

func queryCowinForStates(c *CowinClient, request *http.Request) (*CowinStatesResponse, error) {
	var cowinstateresponse CowinStatesResponse
	path := "/api/v2/admin/location/states"
	responseBytes, err := c.DoJSON(path, request, cowinstateresponse)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(responseBytes, &cowinstateresponse)
	if err != nil {
		return nil, err
	}

	return &cowinstateresponse, nil
}

type States struct {
	StateID   int
	StateName string
}

func (c *CowinClient) GetStates(request *http.Request) []States {
	response, err := queryCowinForStates(c, request)
	if err != nil {
		log.Println("Error: ", err)
	}

	var states []States
	for _, value := range response.States {
		var s States
		s.StateID = value.StateID
		s.StateName = value.StateName
		states = append(states, s)
	}

	return states
}

type CowinDistrictsResponse struct {
	Districts []struct {
		StateID      int    `json:"state_id"`
		DistrictID   int    `json:"district_id"`
		DistrictName string `json:"district_name"`
	} `json:"districts"`
}

type Districts struct {
	StateID      int
	DistrictID   int
	DistrictName string
}

func queryCowinForDistricts(c *CowinClient, request *http.Request, stateID string) (*CowinDistrictsResponse, error) {
	var cowindistrictresponse CowinDistrictsResponse
	path := "/api/v2/admin/location/districts/" + stateID
	responseBytes, err := c.DoJSON(path, request, cowindistrictresponse)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(responseBytes, &cowindistrictresponse)
	if err != nil {
		return nil, err
	}

	return &cowindistrictresponse, nil
}

func (c *CowinClient) GetDistricts(request *http.Request, stateID string) []Districts {
	cowinResponse, err := queryCowinForDistricts(c, request, stateID)
	var districts []Districts
	if err != nil {
		log.Println("ERROR: ", err)
	}
	for _, value := range cowinResponse.Districts {
		var d Districts
		d.StateID = value.StateID
		d.DistrictID = value.DistrictID
		d.DistrictName = strings.ToLower(value.DistrictName)
		districts = append(districts, d)
	}

	return districts
}

type CowinHospitalsResponse struct {
	Centers []struct {
		Name         string `json:"name"`
		Address      string `json:"address"`
		StateName    string `json:"state_name"`
		DistrictName string `json:"district_name"`
		BlockName    string `json:"block_name"`
		Pincode      int    `json:"pincode"`
		Lat          int    `json:"lat"`
		Long         int    `json:"long"`
		Sessions     []struct {
			Date              string  `json:"date"`
			AvailableCapacity float64 `json:"available_capacity"`
			MinAgeLimit       int     `json:"min_age_limit"`
			Vaccine           string  `json:"vaccine"`
		} `json:"sessions"`
	} `json:"centers"`
}

func queryCowinForHospitals(c *CowinClient, request *http.Request, districtID, date string) (*CowinHospitalsResponse, error) {
	var cowinhospitalresponse CowinHospitalsResponse
	path := "/api/v2/appointment/sessions/public/calendarByDistrict"
	c.baseURL.RawQuery = c.addQueryParameters(districtID, date)
	responseBytes, err := c.DoJSON(path, request, cowinhospitalresponse)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(responseBytes, &cowinhospitalresponse)
	if err != nil {
		return nil, err
	}
	return &cowinhospitalresponse, nil
}

type Hospitals struct {
	gorm.Model
	Name              string
	StateName         string
	DistrictName      string
	BlockName         string
	Pincode           int
	Lat               int
	Long              int
	Date              string
	AvailableCapacity float64
	MinAgeLimit       int
	Vaccine           string
}

func (c *CowinClient) GetHospitals(request *http.Request, districtID, date string) []Hospitals {
	cowinResponse, err := queryCowinForHospitals(c, request, districtID, date)
	var hospitals []Hospitals
	if err != nil {
		log.Println("ERROR: ", err)
	}

	if len(cowinResponse.Centers) > 0 {
	for _, value := range cowinResponse.Centers {
			for _, session := range value.Sessions {
				if int(session.AvailableCapacity) > 0 {
					var h Hospitals
					h.Name = value.Name
					h.StateName = value.StateName
					h.DistrictName = value.DistrictName
					h.BlockName = value.BlockName
					h.Pincode = value.Pincode
					h.Lat = value.Lat
					h.Long = value.Long
					h.Date = session.Date
					h.AvailableCapacity = float64(session.AvailableCapacity)
					h.MinAgeLimit = int(session.MinAgeLimit)
					h.Vaccine = session.Vaccine
					hospitals = append(hospitals, h)
				}
			}
		}
	} else {
		fmt.Errorf("ERROR: No records in cowinResponse.Centers")
		return nil
	}

	return hospitals
}
