package scheduler

import (
	// "context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/gammazero/workerpool"

	model "github.com/cowin-slot-checker/cowin-slot-checker-worker/src/model"
)

type CowinStates struct {
	cowinstates model.CowinStatesResponse
}

var refreshStatesTask bool
var stateCodes []string

func (states CowinStates) getStateCodes() ([]string, error) {
	if len(states.cowinstates.States) == 0 {
		return nil, fmt.Errorf("%q", "input dataset for states is empty")
	}

	var codes []string
	for _, value := range states.cowinstates.States {
		codes = append(codes, strconv.Itoa(value.StateID))
	}

	return codes, nil
}

func RefreshStatesTask() {
	start := time.Now()
	refreshStatesTask = false
	log.Println("Refreshing States data")
	client, err := model.NewCowinClient()
	if err != nil {
		fmt.Errorf("ERROR: %q", err)
	}

	urlvalue, err := getParsedURL("https://cdn-api.co-vin.in/api/v2/admin/location/states")
	if err != nil {
		fmt.Errorf("ERROR: ", err)
	}

	req, err := client.NewRequest(http.MethodGet, urlvalue, nil)
	if err != nil {
		fmt.Errorf("ERROR: ", err)
	}

	var cowinstates CowinStates
	cowinstates.cowinstates, err = client.GetStates(req)
	if err != nil {
		fmt.Errorf("ERROR:", err)
	}

	stateCodes, err = cowinstates.getStateCodes()
	if err != nil {
		fmt.Errorf("ERROR:", err)
	}

	log.Println("Refresh of states completed")
	log.Println("Function to refresh states sleeping for 1 month")
	log.Println("Refresh States completed in: ", time.Since(start))
	refreshStatesTask = true
}

var mu sync.Mutex
var refreshDistricts bool

type CowinDistricts struct {
	cowindistricts model.CowinDistrictsResponse
}

var districtCodes []string

func (districts CowinDistricts) getDistrictCodes() ([]string, error) {
	if len(districts.cowindistricts.Districts) == 0 {
		return nil, fmt.Errorf("%q", "input dataset for districts is empty")
	}

	var codes []string
	for _, value := range districts.cowindistricts.Districts {
		codes = append(codes, strconv.Itoa(value.DistrictID))
	}

	return codes, nil
}

func RefreshDistrictsTask() {
	if refreshStatesTask {
		log.Println("Refreshing Districts Data")
		refreshDistricts = false
		start := time.Now()
		client, err := model.NewCowinClient()
		if err != nil {
			fmt.Errorf("ERROR: %q", err)
		}

		wp := workerpool.New(5)
		for _, code := range stateCodes {
			urlValue, err := getParsedURL("https://cdn-api.co-vin.in/api/v2/admin/location/districts/" + code)
			if err != nil {
				fmt.Errorf("ERROR: %q", err)
			}

			req, err := client.NewRequest(http.MethodGet, urlValue, nil)
			if err != nil {
				fmt.Errorf("ERROR: %q", err)
			}

			wp.Submit(func() {
				var districts CowinDistricts
				districts.cowindistricts, err = client.GetDistricts(req, code)
				if err != nil {
					fmt.Println(err)
				}

				value, err := districts.getDistrictCodes()
				if err != nil {
					fmt.Println(err)
				}

				mu.Lock()
				districtCodes = append(districtCodes, value...)
				mu.Unlock()
			})
		}

		wp.StopWait()
		log.Println("Code completed in: ", time.Since(start))
		refreshDistricts = true
		log.Println("Refreshing Districts Data Completed")
	}
}

type CowinHospitals struct {
	cowinhospitals model.CowinHospitalsResponse
}

func (hdata CowinHospitals) getHospitalDetails() ([]Hospital, error) {
	if len(hdata.cowinhospitals.Centers) == 0 {
		return nil, fmt.Errorf("%q", "input dataset for hospitals is empty")
	}

	var hospitalList []Hospital
	for _, value := range hdata.cowinhospitals.Centers {
		var h Hospital
		for _, hospital := range value.Sessions {
			h.Name = value.Name
			h.District = value.DistrictName
			h.BlockName = value.BlockName
			h.StateName = value.StateName
			h.Pincode = value.Pincode
			h.Date = hospital.Date
			h.AvailableCapacity = hospital.AvailableCapacity
			h.MinAgeLimit = hospital.MinAgeLimit
			h.Vaccine = hospital.Vaccine
			h.AvailableCapacityDose1 = hospital.AvailableCapacityDose1
			h.AvailableCapacityDose2 = hospital.AvailableCapacityDose2
			hospitalList = append(hospitalList, h)
		}
	}

	return hospitalList, nil
}

type Hospitals struct {
	hospitals [][]Hospital
}

var refreshHospitals bool
var hdata Hospitals

func RefreshHospitalsTask() {
	refreshHospitals = false
	if refreshDistricts {
		start := time.Now()
		log.Println("Refreshing Hospitals Data")
		client, err := model.NewCowinClient()
		if err != nil {
			log.Println(err)
		}

		date := time.Now().Format("02-01-2006")
		wp := workerpool.New(5)
		for _, district := range districtCodes {
			urlValue, err := url.Parse("https://cdn-api.co-vin.in/api/v2/appointment/sessions/public/calendarByDistrict?date=" + date + "&" + "district_id=" + district)
			if err != nil {
				log.Println("ERROR: ", err)
			}

			request, err := client.NewRequest(http.MethodGet, urlValue, nil)
			if err != nil {
				log.Println("ERROR: ", err)
			}

			wp.Submit(func() {
				var cowinHospitals CowinHospitals
				cowinHospitals.cowinhospitals, err = client.GetHospitals(request, district)
				if err != nil {
					fmt.Println(err)
				}

				data, err := cowinHospitals.getHospitalDetails()
				if err != nil {
					fmt.Println(err)
				}

				mu.Lock()
				if len(data) > 0 {
					hdata.hospitals = append(hdata.hospitals, data)
				}
				mu.Unlock()
			})
		}

		wp.StopWait()
		log.Println("Refresh of Hospitals completed")
		log.Println("Function to refresh hospitals sleeping for 30 minutes")
		log.Println("Code completed in: ", time.Since(start))
	}
	refreshHospitals = true
}
