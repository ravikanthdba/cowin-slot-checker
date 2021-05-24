package scheduler

import (
	"fmt"
	"log"
	"strconv"
	"time"

	cache "github.com/cowin-slot-checker/cowin-slot-checker-worker/src/cache"
	database "github.com/cowin-slot-checker/cowin-slot-checker-worker/src/database"
)

type HospitalByPincode struct {
	Pincode  string
	Hospital []Hospital
}

func CacheRefreshTask() {
	if refreshDatabase {
		log.Println("Launching Cache Refresh Task")
		start := time.Now()
		db, err := database.CreateConnection("cowin_user", "GenieHello^123", "172.23.227.40", "cowin_database")
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

		data, err := db.DatabaseQueryWithoutCondition("select distinct hospitals.name as hospital,  hospitals.district as district, hospitals.block_name as block,  hospitals.date as date, hospitals.available_capacity as available_capacity, hospitals.min_age_limit as min_age_limit, hospitals.vaccine as vaccine, hospitals.pincode as pincode, hospitals.available_capacity_dose1 as dose1, hospitals.available_capacity_dose2 as dose2 from hospitals join postoffices on (hospitals.pincode = postoffices.pincode) where hospitals.deleted_at is null  and postoffices.deleted_at is null and hospitals.date between date_format(now(), '%d-%m-%Y') and  date_format(date_add(now(), interval 3 day), '%d-%m-%Y') and available_capacity > 0;")
		if err != nil {
			fmt.Errorf("%q", err)
		}

		records, err := data.Rows()
		if err != nil {
			fmt.Errorf("%q", err)
		}

		// var city []HospitalByPincode

		redisClient, err := cache.CreateConnection("localhost", "", 6379)
		if err != nil {
			fmt.Errorf("%q", err)
		}

		rh, err := cache.NewReJSONHandler(redisClient)
		if err != nil {
			fmt.Errorf("%q", err)
		}

		for records.Next() {
			var hospital, districtName, blockName, date, vaccine string
			var availableCapacity, minAgeLimit, availableCapacityDose1, availableCapacityDose2, pincode float64
			if err := records.Scan(&hospital, &districtName, &blockName, &date, &availableCapacity, &minAgeLimit, &vaccine, &pincode, &availableCapacityDose1, &availableCapacityDose2); err != nil {
				log.Println("ERROR: ", err)
			}

			var hospitalData Hospital
			var district HospitalByPincode
			district.Pincode = strconv.Itoa(int(pincode))
			hospitalData.District = districtName
			hospitalData.Name = hospital
			hospitalData.BlockName = blockName
			hospitalData.Date = date
			hospitalData.AvailableCapacity = availableCapacity
			hospitalData.MinAgeLimit = minAgeLimit
			hospitalData.Vaccine = vaccine
			hospitalData.Pincode = pincode
			hospitalData.AvailableCapacityDose1 = availableCapacityDose1
			hospitalData.AvailableCapacityDose2 = availableCapacityDose2
			district.Hospital = append(district.Hospital, hospitalData)
			var interfaceData []interface{}
			for _, value := range district.Hospital {
				interfaceData = append(interfaceData, value)
			}
			log.Println("Setting Cache in Redis for pincode: ", district.Pincode)
			_, err = rh.Set("cowin", district.Pincode, interfaceData)
			if err != nil {
				fmt.Errorf("%q", err)
			}
		}

		log.Println("Cache Refresh Task Completed in: ", time.Since(start))
	}
}
