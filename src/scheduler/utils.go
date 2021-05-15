package scheduler

import (
	"strconv"
	"fmt"
	"net/url"
	"time"

	model "github.com/cowin-slot-checker/src/model"
)

const (
	format = "2006-01-02 15:04:05"
)

func getStateCodes(states []model.States) ([]string, error) {
	if len(states) == 0 {
		return nil, fmt.Errorf("No input passed into getStateCodes")
	}

	var stateCodes []string
	for _, value := range states {
		stateCodes = append(stateCodes, strconv.Itoa(value.StateID))
	}
	return stateCodes, nil
}

func getParsedURL(u string) (*url.URL, error) {
	urlValue, err := url.Parse(u)
	if err != nil {
		return nil, fmt.Errorf("ERROR: ", err)
	}
	return urlValue, nil
}

func (h HospitalDistrictMapping) convertToInterfaces() []interface{} {
	var data []interface{}
	for _, value := range h.hospitalsList {
		for _, hospital := range value {
			data = append(data, hospital)
		}
	}
	return data
}

func dateToQuery(date string, timeToLookback time.Duration) string {
	parsedDate, err := time.Parse(format, date)
	if err != nil {
		fmt.Errorf("ERROR: ", err)
	}
	return parsedDate.Add(timeToLookback * time.Minute).Format(format)
}