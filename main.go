package main

import (
	"fmt"
	"log"
	"net/http"

	scheduler "scheduler"

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
		scheduler.RefreshHospitalsTask()
		scheduler.DatabaseRefreshTask()
	}()

	schedule := cron.New()
	log.Println("Scheduling the background jobs")
	schedule.AddFunc("@every 720h", scheduler.RefreshStatesTask)
	schedule.AddFunc("@every 360h", scheduler.RefreshDistrictsTask)
	schedule.AddFunc("@every 20m", scheduler.RefreshHospitalsTask)
	schedule.AddFunc("@every 30m", scheduler.DatabaseRefreshTask)
	log.Println("Scheduling Completed")
	schedule.Start()

	// bot, err := tgbotapi.NewBotAPI("1723271445:AAFSMcBwLq70nomEAWgYTj_D6ItqsCGaWz4")
	// if err != nil {
	// 	log.Panic(err)
	// }

	// bot.Debug = true

	// log.Printf("Authorized on account %s", bot.Self.FirstName)

	// u := tgbotapi.NewUpdate(0)
	// u.Timeout = 60

	// updates, err := bot.GetUpdatesChan(u)

	// for update := range updates {
	// 	if update.Message == nil { // ignore any non-Message Updates
	// 		continue
	// 	}

	// 	log.Printf("[%s]:  %s", update.Message.From, update.Message.Text)
	// 	var message tgbotapi.MessageConfig
	// 	if update.Message.Text == "why" {
	// 		message = tgbotapi.NewMessage(update.Message.Chat.ID, "This is covin app tracker")
	// 	} else {
	// 		message = tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text+update.Message.From.String())
	// 	}

	// 	message.ReplyToMessageID = update.Message.MessageID
	// 	log.Println("====")
	// 	bot.Send(message)
	// }
	http.HandleFunc("/status", Hello)
	http.ListenAndServe(":8080", nil)
}
