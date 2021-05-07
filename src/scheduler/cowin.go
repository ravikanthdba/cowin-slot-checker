package scheduler

import (
	"context"
	"log"
	model "model"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/gammazero/workerpool"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	bulk "github.com/t-tiger/gorm-bulk-insert"
)

func GetStateCodes(conn *model.CowinClient) []string {
	urlValue, err := url.ParseRequestURI("https://cdn-api.co-vin.in/api/v2/admin/location/states")
	if err != nil {
		log.Println("ERROR:", err)
	}
	request, err := conn.NewRequest(context.Background(), http.MethodGet, urlValue, nil)
	if err != nil {
		log.Println("Error: ", err)
	}

	response := conn.GetStates(request)

	var stateID []string
	for _, value := range response {
		stateID = append(stateID, strconv.Itoa(value.StateID))
	}

	return stateID
}

var stateCodes []string
var refreshStatesTask bool

func RefreshStatesTask() {
	start := time.Now()
	refreshStatesTask = false
	client, err := model.NewClient(nil)
	if err != nil {
		log.Println(err)
	}
	log.Println("Refreshing States data")
	stateCodes = GetStateCodes(client)
	log.Println("Refresh of states completed")
	log.Println("Function to refresh states sleeping for 1 month")
	log.Println("Refresh States completed in: ", time.Now().Sub(start))
	refreshStatesTask = true
}

func GetDistrictCodes(conn *model.CowinClient, stateID string, urlValue *url.URL) []model.Districts {
	request, err := conn.NewRequest(context.Background(), http.MethodGet, urlValue, nil)
	if err != nil {
		log.Println("Error:", err)
	}

	response := conn.GetDistricts(request, stateID)
	return response
}

type Mapping struct {
	mu sync.Mutex
	d  []string
}

var c Mapping
var refreshDistrictTask bool

func RefreshDistrictsTask() {
	if refreshStatesTask {
		refreshDistrictTask = false
		client, err := model.NewClient(nil)
		if err != nil {
			log.Println(err)
		}

		start := time.Now()
		log.Println("Refreshing Districts data")
		done := make(chan bool, len(stateCodes))
		for _, code := range stateCodes {
			go func(code string) {
				log.Println("Launching query for district code: ", code)
				urlValue, _ := url.ParseRequestURI("https://cdn-api.co-vin.in/api/v2/admin/location/districts/" + code)
				districts := GetDistrictCodes(client, code, urlValue)
				for _, code := range districts {
					c.mu.Lock()
					c.d = append(c.d, strconv.Itoa(code.DistrictID))
					c.mu.Unlock()
				}
				done <- true
			}(code)
		}

		for i := 1; i <= len(stateCodes); i++ {
			<-done
		}
		log.Println("Refresh of districts completed")
		log.Println("Function to refresh districts sleeping for 15 days")
		log.Println("Refresh Districts completed in: ", time.Now().Sub(start))
		refreshDistrictTask = true
	}
}

func worker(conn *model.CowinClient, request *http.Request, id, date string) []model.Hospitals {
	var hospitalResponse []model.Hospitals
	log.Println("worker for district id: ", id, " started")
	hospitalResponse = conn.GetHospitals(request, id, date)
	log.Println("worker for district id: ", id, " finished")
	return hospitalResponse
}

type HospitalDistrictMapping struct {
	mu        sync.Mutex
	hospitals [][]model.Hospitals
}

var hdata HospitalDistrictMapping
var refreshHospitals bool

func RefreshHospitalsTask() {
	refreshHospitals = false
	if refreshStatesTask {
		log.Println("Refreshing Hospitals Data")
		client, err := model.NewClient(nil)
		if err != nil {
			log.Println(err)
		}
		wp := workerpool.New(5)
		requests := c.d
		start := time.Now()
		for _, r := range requests {
			r := r
			wp.Submit(func() {
				date := time.Now().Format("02-01-2006")
				urlValue, _ := url.ParseRequestURI("https://cdn-api.co-vin.in/api/v2/appointment/sessions/public/calendarByDistrict?date=" + date + "&" + "district_id=" + r)
				request, err := client.NewRequest(context.Background(), http.MethodGet, urlValue, nil)
				if err != nil {
					log.Println("ERROR: ", err)
				}
				hresponse := worker(client, request, r, date)
				hdata.mu.Lock()
				if len(hresponse) > 0 {
					hdata.hospitals = append(hdata.hospitals, hresponse)
				}
				hdata.mu.Unlock()
			})
		}

		wp.StopWait()
		log.Println("Refresh of Hospitals completed")
		log.Println("Function to refresh hospitals sleeping for 30 minutes")
		log.Println("Code completed in: ", time.Now().Sub(start))
	}
	refreshHospitals = true
}

func (h HospitalDistrictMapping) convertToInterfaces() []interface{} {
	var data []interface{}
	for _, value := range h.hospitals {
		for _, hospital := range value {
			data = append(data, hospital)
		}
	}
	return data
}

func DatabaseRefreshTask() {
	if refreshHospitals {
		start := time.Now()
		db, err := gorm.Open("mysql", "cowin:GenieHello123*@tcp(172.23.227.40:3306)/cowin_database")
		if err != nil {
			log.Println("ERROR: Connecting to database: ", err)
		}
		log.Println("Connection Established")
		defer db.Close()
		db.AutoMigrate(&model.Hospitals{})
		data := hdata.convertToInterfaces()
		bulk.BulkInsert(db, data, 300)
		log.Println("Insert into the database has been completed")
		deleteTime := time.Now().Add(-40 * time.Minute).Format("2006-01-02 15:04:05")
		log.Println("Soft Deleting all data before ", deleteTime)
		db.Unscoped().Where("created_at > ?", deleteTime).Delete(&model.Hospitals{})
		log.Println("Delete Completed")
		db.Commit()
		log.Println("Database Transaction completed in: ", time.Now().Sub(start))
	}
}
