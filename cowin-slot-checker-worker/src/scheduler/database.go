package scheduler

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/cowin-slot-checker/cowin-slot-checker-worker/src/database"
)

var refreshDatabase bool

func PostalCodeDatabaseRefreshTask() {
	log.Println("Refreshing Postal Code Database Data")
	start := time.Now()
	refreshDatabase = false
	db, err := database.CreateConnection("cowin_user", "GenieHello^123", "172.23.227.40", "cowin_database")
	if err != nil {
		fmt.Errorf("ERROR: %q", err)
	}
	defer db.Connection.Close()

	log.Println("Creating Tables if not exists")
	tables, err := db.AutoMigrateTables(Postoffice{})
	if !tables {
		fmt.Errorf("%q", err)
	}

	log.Println("Bulk loading data into the table")
	interfaceData := interfaceConversion(pdata)
	_, err = db.Load(interfaceData, 500)
	if err != nil {
		fmt.Errorf("%q", err)
	}

	results, err := db.DatabaseQueryWithoutCondition("select max(created_at) from postoffices")
	if err != nil {
		fmt.Errorf("ERROR: Error Querying database: %q", err)
	}

	createdAt, err := results.Rows()
	if err != nil {
		fmt.Errorf("%q", err)
	}

	var date string
	for createdAt.Next() {
		if err := createdAt.Scan(&date); err != nil {
			panic(err)
		}
	}
	log.Println("Records before ", dateToQuery(date, -20), " will be deleted")

	_, err = db.DeleteRecords("created_at < ?", &Postoffice{}, dateToQuery(date, -20))
	if err != nil {
		fmt.Errorf("ERROR: Error Querying database: %q", err)
	}

	var recordCount []int
	countRecords, err := db.DatabaseQueryWithCondition("select count(*) from postoffices where deleted_at is not null and created_at < ?", &recordCount, dateToQuery(date, -20))
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
	log.Println("Postal Code Database Refresh completed in: ", time.Since(start))
}

func HospitalsDatabaseRefreshTask() {
	log.Println("Refreshing Database Data")
	start := time.Now()
	refreshDatabase = false
	db, err := database.CreateConnection("cowin_user", "GenieHello^123", "172.23.227.40", "cowin_database")
	if err != nil {
		fmt.Errorf("ERROR: ", err)
	}
	defer db.Connection.Close()

	log.Println("Creating Tables if not exists")
	tables, err := db.AutoMigrateTables(Hospital{})
	if !tables {
		fmt.Errorf("%q", err)
	}

	log.Println("Bulk loading data into the table")
	interfaceData := interfaceConversion(hdata)
	_, err = db.Load(interfaceData, 500)
	if err != nil {
		fmt.Errorf("%q", err)
	}
	results, err := db.DatabaseQueryWithoutCondition("select max(created_at) from hospitals")
	if err != nil {
		fmt.Errorf("ERROR: Error Querying database: %q", err)
	}

	createdAt, err := results.Rows()
	if err != nil {
		fmt.Errorf("%q", err)
	}

	var date string
	for createdAt.Next() {
		if err := createdAt.Scan(&date); err != nil {
			panic(err)
		}
	}
	log.Println("Records before ", dateToQuery(date, -20), " will be deleted")

	_, err = db.DeleteRecords("created_at < ?", &Hospital{}, dateToQuery(date, -20))
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
	log.Println("Hospitals Database Refresh completed in: ", time.Since(start))
}
