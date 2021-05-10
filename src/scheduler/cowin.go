package scheduler

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
	"flag"
	"fmt"
	"github.com/gammazero/workerpool"
	"github.com/jinzhu/gorm"
	rejson "github.com/nitishm/go-rejson/v4"
	goredis "github.com/go-redis/redis/v8"
	model "github.com/cowin-slot-checker/src/model"
	database "github.com/cowin-slot-checker/src/database"
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
	hospitalResponse = conn.GetHospitals(request, id, date)
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

var refreshDatabase bool

func dateToQuery(conn *gorm.DB, table string, timeToLookback time.Duration) string {
	var minback, maxday string
	maxDate, _ := conn.Table("hospitals").Select("max(created_at) as date").Rows()
	for maxDate.Next() {
		if err := maxDate.Scan(&maxday); err != nil {
			log.Fatal(err)
		}
		modtime, err := time.Parse("2006-01-02 15:04:05", maxday)
		if err != nil {
			log.Fatal(err)
		}
		minback = modtime.Add(timeToLookback * time.Minute).Format("2006-01-02 15:04:05")
	}
	return minback
}

func DatabaseRefreshTask() {
	if refreshHospitals {
		refreshDatabase = false
		start := time.Now()
		log.Println("Database Connection Established for database Refresh")
		db, err := database.Connection("cowin", "HappyVaccination123*", "localhost", "cowin_database")
		if err != nil {
			log.Println(err)
		}
		defer db.Connection.Close()
		db.Connection.AutoMigrate(&model.Hospitals{})
		data := hdata.convertToInterfaces()
		_, err = db.Load(data, 300)
		if err != nil {
			log.Println("ERROR: bulk load has failed: ", err)
			return
		}
		log.Println("Insert into the database has been completed")



		deletiondate := dateToQuery(db.Connection, "hospital", -30)
		log.Println("Hard Deleting all data before ", deletiondate)
		var hospitals model.Hospitals
		tx := db.Connection.Begin()
		results := db.Connection.Where("created_at < ?", deletiondate).Delete(&hospitals)
		tx.Commit()
		if results.Error != nil {
			log.Println("ERROR: ", results.Error)
			return
		}
		log.Println("Deleted " + strconv.Itoa(int(results.RowsAffected)) + " Records.")
		log.Println("Database Transaction completed in: ", time.Now().Sub(start))
	}
	refreshDatabase = true
}

type Result struct {
	Name string
	BlockName string
	Date string
	AvailableCapacity float64
	MinAgeLimit float64
	Vaccine string
}

type Results struct {
	DistrictName string
	Result Result
}

func UpdateCache(rh *rejson.Handler, finalResult []Results) {
	res, err := rh.JSONSet("cowinResult", ".", finalResult)
	if err != nil {
		log.Fatalf("Failed to JSONSet")
		return
	}

	if res.(string) == "OK" {
		fmt.Printf("Success: %s\n", res)
	} else {
		fmt.Println("Failed to Set: ")
	}
}

func CacheRefreshTask() {
		if refreshDatabase {
			log.Println("Launching Cache Refresh Task")
			db, err := database.Connection("cowin", "HappyVaccination123*", "localhost", "cowin_database")
			if err != nil {
				log.Println(err)
			}
			defer db.Connection.Close()
			querydate := dateToQuery(db.Connection, "hospital", -10)
			log.Println("Querying database for fetching data")
			var finalResult []Results
			tx := db.Connection.Begin()
			records, _ := tx.Table("hospitals").Select("name, district_name, block_name, date, available_capacity, min_age_limit, vaccine").Where("created_at > ?", querydate).Rows()
			for records.Next() {
				var name, districtName, blockName, date, vaccine string
				var availableCapacity, minAgeLimit float64
				if err := records.Scan(&name, &districtName, &blockName, &date, &availableCapacity, &minAgeLimit, &vaccine); err != nil {
					log.Println("ERROR: ", err)
				}
				var results Results
				
				results.DistrictName = districtName
				results.Result.Name = name
				results.Result.BlockName = blockName
				results.Result.Date = date
				results.Result.AvailableCapacity = availableCapacity
				results.Result.MinAgeLimit = minAgeLimit
				results.Result.Vaccine = vaccine
				finalResult = append(finalResult, results)
			}
			tx.Commit()
			log.Println("Fetching Records Completed")
			log.Println("Setting Cache in Redis")
			var addr = flag.String("Server", "localhost:6379", "Redis server address")
			rh := rejson.NewReJSONHandler()
			flag.Parse()
			cli := goredis.NewClient(&goredis.Options{Addr: *addr})
			rh.SetGoRedisClient(cli)
			UpdateCache(rh, finalResult)
		}
}