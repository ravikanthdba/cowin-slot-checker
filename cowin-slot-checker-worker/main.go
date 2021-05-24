package main

import (
	"fmt"
	"log"
	"net/http"

	scheduler "github.com/cowin-slot-checker/cowin-slot-checker-worker/src/scheduler"
	"github.com/robfig/cron"
)

func Status(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "GOOD")
}

func main() {
	go func() {
		log.Println("Launching Background Processes to Refresh data after application restarts")
		scheduler.RefreshStatesTask()
		scheduler.RefreshDistrictsTask()
		scheduler.RefreshHospitalsTask()
		scheduler.HospitalsDatabaseRefreshTask()
		scheduler.CapturePostalCodesTask()
		scheduler.PostalCodeDatabaseRefreshTask()
		// scheduler.CachingDistrictsTask()
		scheduler.CacheRefreshTask()
	}()

	schedule := cron.New()
	log.Println("Scheduling the background jobs")
	schedule.AddFunc("@every 720h", scheduler.RefreshStatesTask)
	schedule.AddFunc("@every 360h", scheduler.RefreshDistrictsTask)
	schedule.AddFunc("@every 20m", scheduler.RefreshHospitalsTask)
	schedule.AddFunc("@every 30m", scheduler.CapturePostalCodesTask)
	schedule.AddFunc("@every 40m", scheduler.PostalCodeDatabaseRefreshTask)
	schedule.AddFunc("@every 5m", scheduler.CacheRefreshTask)
	log.Println("Scheduling Completed")
	schedule.Start()

	http.HandleFunc("/status", Status)
	http.ListenAndServe(":8080", nil)
}
