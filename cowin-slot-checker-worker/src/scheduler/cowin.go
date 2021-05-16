package scheduler

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"strconv"
	"sync"
	"time"

	"github.com/gammazero/workerpool"

	cache "github.com/cowin-slot-checker/cowin-slot-checker-worker/src/cache"
	database "github.com/cowin-slot-checker/cowin-slot-checker-worker/src/database"
	model "github.com/cowin-slot-checker/cowin-slot-checker-worker/src/model"
)

var refreshStatesTask bool
var stateCodes []string

func RefreshStatesTask() {
	start := time.Now()
	refreshStatesTask = false
	log.Println("Refreshing States data")
	client, err := model.NewClient(nil)
	if err != nil {
		fmt.Errorf("ERROR: %q", err)
	}
	urlvalue, err := getParsedURL("https://cdn-api.co-vin.in/api/v2/admin/location/states")
	if err != nil {
		fmt.Errorf("ERROR: ", err)
	}
	req, err := client.NewRequest(context.Background(), http.MethodGet, urlvalue, nil)
	if err != nil {
		fmt.Errorf("ERROR: ", err)
	}
	states, err := client.GetStates(req)
	if err != nil {
		fmt.Errorf("ERROR:", err)
	}
	if stateCodes, err = getStateCodes(states); err != nil {
		fmt.Errorf("ERROR:", err)
	}
	log.Println("Refresh of states completed")
	log.Println("Function to refresh states sleeping for 1 month")
	log.Println("Refresh States completed in: ", time.Since(start))
	refreshStatesTask = true
}

var mu sync.Mutex
var refreshDistricts bool

type DistrictsData struct {
	Districts [][]model.Districts
}

var districtsData DistrictsData

func RefreshDistrictsTask() {
	if refreshStatesTask {
		log.Println("Refreshing Districts Data")
		refreshDistricts = false
		start := time.Now()
		client, err := model.NewClient(nil)
		if err != nil {
			log.Println("ERROR: ", err)
		}
		wp := workerpool.New(5)
		for _, code := range stateCodes {
			urlValue, err := getParsedURL("https://cdn-api.co-vin.in/api/v2/admin/location/districts/" + code)
			if err != nil {
				fmt.Errorf("ERROR: ", err)
			}
			req, err := client.NewRequest(context.Background(), http.MethodGet, urlValue, nil)
			if err != nil {
				fmt.Errorf("ERROR: ", err)
			}

			wp.Submit(func() {
				districts := client.GetDistricts(req, code)
				mu.Lock()
				districtsData.Districts = append(districtsData.Districts, districts)
				mu.Unlock()
			})
		}
		wp.StopWait()
		log.Println("Code completed in: ", time.Since(start))
		refreshDistricts = true
		log.Println("Refreshing Districts Data Completed")
	}
}

func CachingDistrictsTask() {
	if refreshDistricts {
		start := time.Now()
		redisClient, err := cache.CreateConnection("20.197.29.58", "", 6379)
		if err != nil {
			fmt.Errorf("%q", err)
		}

		rh, err := cache.NewReJSONHandler(redisClient)
		if err != nil {
			fmt.Errorf("%q", err)
		}

		var interfaceData []interface{}
		for _, value := range districtsData.Districts {
			interfaceData = append(interfaceData, value)
		}
		_, err = rh.Set("districts", ".", interfaceData)
		if err != nil {
			fmt.Errorf("%q", err)
		}
		log.Println("Cache Refresh Task Completed in: ", time.Since(start))
	}
}

type HospitalDistrictMapping struct {
	hospitalsList [][]model.Hospitals
}

var hdata HospitalDistrictMapping
var refreshHospitals bool

func RefreshHospitalsTask() {
	refreshHospitals = false
	if refreshDistricts {
		start := time.Now()
		log.Println("Refreshing Hospitals Data")
		client, err := model.NewClient(nil)
		if err != nil {
			log.Println(err)
		}
		wp := workerpool.New(5)
		districtcodes, err := districtsData.getDistrictCodes()
		if err != nil {
			log.Println(err)
		}

		date := start.Format("02-01-2006")
		for _, district := range districtcodes {
			urlValue, err := getParsedURL("https://cdn-api.co-vin.in/api/v2/appointment/sessions/public/calendarByDistrict?date=" + date + "&" + "district_id=" + district)
			if err != nil {
				log.Println("ERROR: ", err)
			}
			request, err := client.NewRequest(context.Background(), http.MethodGet, urlValue, nil)
			if err != nil {
				log.Println("ERROR: ", err)
			}
			wp.Submit(func() {
				hospitalResponse := client.GetHospitals(request, district, date)
				mu.Lock()
				if len(hospitalResponse) > 0 {
					hdata.hospitalsList = append(hdata.hospitalsList, hospitalResponse)
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

var refreshDatabase bool

func DatabaseRefreshTask() {
	if refreshHospitals {
		log.Println("Refreshing Database Data")
		start := time.Now()
		refreshDatabase = false
		db, err := database.CreateConnection("cowin_user", "HappyVaccination123$", "localhost", "cowin_database")
		if err != nil {
			fmt.Errorf("ERROR: ", err)
		}
		defer db.Connection.Close()
		log.Println("Creating Tables if not exists")
		autoMigrateResults, err := db.AutoMigrateTables(model.Hospitals{})
		if !autoMigrateResults {
			fmt.Errorf("%q", err)
		}

		log.Println("Bulk loading data into the table")
		interfaceData := hdata.convertToInterfaces()
		_, err = db.Load(interfaceData, 500)
		if err != nil {
			fmt.Errorf("%q", err)
		}

		results, err := db.DatabaseQueryWithoutCondition("select max(created_at) from hospitals")
		if err != nil {
			fmt.Errorf("ERROR: Error Querying database: %q", err)
		}
		var date string
		createdAt, err := results.Rows()
		if err != nil {
			fmt.Errorf("%q", err)
		}
		for createdAt.Next() {
			if err := createdAt.Scan(&date); err != nil {
				fmt.Errorf("%q", err)
			}
		}
		log.Println("Records before ", dateToQuery(date, -20), " will be deleted")

		_, err = db.DeleteRecords("created_at < ?", &model.Hospitals{}, dateToQuery(date, -20))
		if err != nil {
			fmt.Errorf("ERROR: Error Querying database: %q", err)
		}

		var recordCount []int
		countRecords, err := db.DatabaseQueryWithCondition("select count(*) from hospitals where deleted_at is not null and created_at < ?", &recordCount, dateToQuery(date, -20))
		if err != nil {
			fmt.Errorf("ERROR: Error Querying database: %q", err)
		}

		deleteCount, err := countRecords.Rows()
		if err != nil {
			fmt.Errorf("%q", err)
		}
		var count int
		for deleteCount.Next() {
			if err := deleteCount.Scan(&count); err != nil {
				fmt.Errorf("%q", err)
			}
		}
		log.Println(strconv.Itoa(count) + " records have been deleted")
		refreshDatabase = true
		log.Println("Database Refresh completed in: ", time.Since(start))
	}
}

type Hospital struct {
	District          string
	Name              string
	BlockName         string
	Date              string
	AvailableCapacity float64
	MinAgeLimit       float64
	Vaccine           string
}

type District struct {
	DistrictName string
	Hospital     []Hospital
}

func CacheRefreshTask() {
	if refreshDatabase {
		log.Println("Launching Cache Refresh Task")
		start := time.Now()
		db, err := database.CreateConnection("cowin_user", "HappyVaccination123$", "localhost", "cowin_database")
		if err != nil {
			log.Println(err)
		}
		defer db.Connection.Close()

		results, err := db.DatabaseQueryWithoutCondition("select max(created_at) from hospitals")
		if err != nil {
			fmt.Errorf("ERROR: Error Querying database: %q", err)
		}
		var date string
		createdAt, err := results.Rows()
		if err != nil {
			fmt.Errorf("%q", err)
		}
		for createdAt.Next() {
			if err := createdAt.Scan(&date); err != nil {
				fmt.Errorf("%q", err)
			}
		}

		log.Println("Querying for last 5 minutes data, i.e: ", dateToQuery(date, -5))

		data, err := db.DatabaseQueryWithCondition("select name, district_name, block_name, date, available_capacity, min_age_limit, vaccine from hospitals where deleted_at is null and created_at > ?", results, dateToQuery(date, -5))
		if err != nil {
			fmt.Errorf("%q", err)
		}

		records, err := data.Rows()
		if err != nil {
			fmt.Errorf("%q", err)
		}

		var city []District
		for records.Next() {
			var name, districtName, blockName, date, vaccine string
			var availableCapacity, minAgeLimit float64
			if err := records.Scan(&name, &districtName, &blockName, &date, &availableCapacity, &minAgeLimit, &vaccine); err != nil {
				log.Println("ERROR: ", err)
			}

			var hospitalData Hospital
			var district District
			district.DistrictName = districtName
			hospitalData.District = districtName
			hospitalData.Name = name
			hospitalData.BlockName = blockName
			hospitalData.Date = date
			hospitalData.AvailableCapacity = availableCapacity
			hospitalData.MinAgeLimit = minAgeLimit
			hospitalData.Vaccine = vaccine
			district.Hospital = append(district.Hospital, hospitalData)
			city = append(city, district)
		}

		log.Println("Fetching Records Completed")
		log.Println("Setting Cache in Redis")
		redisClient, err := cache.CreateConnection("20.197.29.58", "", 6379)
		if err != nil {
			fmt.Errorf("%q", err)
		}

		rh, err := cache.NewReJSONHandler(redisClient)
		if err != nil {
			fmt.Errorf("%q", err)
		}

		var interfaceData []interface{}
		for _, value := range city {
			interfaceData = append(interfaceData, value)
		}
		_, err = rh.Set("cowin", ".", interfaceData)
		if err != nil {
			fmt.Errorf("%q", err)
		}
		log.Println("Cache Refresh Task Completed in: ", time.Since(start))
	}
}
