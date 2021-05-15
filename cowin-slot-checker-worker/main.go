package main

import (
	"fmt"
	"log"
	"net/http"

	scheduler "github.com/cowin-slot-checker/cowin-slot-checker-worker/src/scheduler"
	"github.com/robfig/cron"
)

func Hello(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "GOOD")
}

func main() {
	go func() {
		log.Println("Launching Background Processes to Refresh data after application restarts")
		scheduler.RefreshStatesTask()
		scheduler.RefreshDistrictsTask()
		scheduler.CachingDistrictsTask()
		scheduler.RefreshHospitalsTask()
		scheduler.DatabaseRefreshTask()
		scheduler.CacheRefreshTask()
	}()

	schedule := cron.New()
	log.Println("Scheduling the background jobs")
	schedule.AddFunc("@every 720h", scheduler.RefreshStatesTask)
	schedule.AddFunc("@every 360h", scheduler.RefreshDistrictsTask)
	schedule.AddFunc("@every 365h", scheduler.CachingDistrictsTask)
	schedule.AddFunc("@every 20m", scheduler.RefreshHospitalsTask)
	schedule.AddFunc("@every 30m", scheduler.DatabaseRefreshTask)
	schedule.AddFunc("@every 5m", scheduler.CacheRefreshTask)
	log.Println("Scheduling Completed")
	schedule.Start()

	http.HandleFunc("/status", Hello)
	http.ListenAndServe(":8080", nil)
}
