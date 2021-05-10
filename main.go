package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/robfig/cron"
	scheduler "github.com/cowin-slot-checker/src/scheduler"
)

func Hello(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "GOOD")
}

func main() {
	go func() {
		log.Println("Launching Background Processes to Refresh data after application restarts")
		scheduler.RefreshStatesTask()
		scheduler.RefreshDistrictsTask()
		scheduler.RefreshHospitalsTask()
		scheduler.DatabaseRefreshTask()
		scheduler.CacheRefreshTask()
	}()

	schedule := cron.New()
	log.Println("Scheduling the background jobs")
	schedule.AddFunc("@every 720h", scheduler.RefreshStatesTask)
	schedule.AddFunc("@every 360h", scheduler.RefreshDistrictsTask)
	schedule.AddFunc("@every 20m", scheduler.RefreshHospitalsTask)
	schedule.AddFunc("@every 30m", scheduler.DatabaseRefreshTask)
	schedule.AddFunc("@every 40m", scheduler.CacheRefreshTask)
	log.Println("Scheduling Completed")
	schedule.Start()

	http.HandleFunc("/status", Hello)
	http.ListenAndServe(":8080", nil)
}
