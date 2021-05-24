package scheduler

import (
	"fmt"
	"net/url"
	"time"
)

const (
	format = "2006-01-02 15:04:05"
)

func getParsedURL(u string) (*url.URL, error) {
	urlValue, err := url.Parse(u)
	if err != nil {
		return nil, fmt.Errorf("ERROR: ", err)
	}
	return urlValue, nil
}

type conversion interface {
	convertToInterfaces() []interface{}
}

func (h Hospitals) convertToInterfaces() []interface{} {
	var records []interface{}
	for _, data := range h.hospitals {
		for _, record := range data {
			records = append(records, record)
		}
	}
	return records
}

func (p PostOffices) convertToInterfaces() []interface{} {
	var records []interface{}
	for _, data := range p.postOffice {
		for _, record := range data {
			records = append(records, record)
		}
	}
	return records
}

func interfaceConversion(c conversion) []interface{} {
	return c.convertToInterfaces()
}

func dateToQuery(date string, timeToLookback time.Duration) string {
	parsedDate, err := time.Parse(format, date)
	if err != nil {
		fmt.Errorf("ERROR: ", err)
	}
	return parsedDate.Add(timeToLookback * time.Minute).Format(format)
}
