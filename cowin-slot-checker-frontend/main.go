package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	cache "github.com/cowin-slot-checker/cowin-slot-checker-worker/src/cache"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/gomodule/redigo/redis"
)

type Hospital struct {
	District          string
	Name              string
	BlockName         string
	Date              string
	AvailableCapacity float64
	MinAgeLimit       float64
	Vaccine           string
}

type HospitalsData struct {
	DistrictName string
	Hospital     []Hospital
}

type Districts struct {
	DistrictID   int
	DistrictName string
}

func districtExists(handler *cache.RedisHandler, districtName string) (string, error) {
	districts, err := redis.Bytes(handler.Get("districts", "."))
	if err != nil {
		return "", err
	}

	var districtsData [][]Districts
	err = json.Unmarshal(districts, &districtsData)
	if err != nil {
		return "", err
	}

	for _, district := range districtsData {
		for _, area := range district {
			if strings.Contains(area.DistrictName, districtName) {
				return area.DistrictName, nil
			}
		}
	}
	return "", fmt.Errorf(districtName + " not exists")
}

func main() {
	bot, err := tgbotapi.NewBotAPI("1894756332:AAEHrsvfk36DeH5AA3D_VPbM5hEfOTURKUQ")
	if err != nil {
		log.Panic(err)
	}

	// bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.FirstName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, _ := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		log.Printf("[%s]:  %s", update.Message.From, update.Message.Text)

		if update.Message.Text == "/start" {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Welcome to Cowin Telegram Chat Bot "+update.Message.Chat.FirstName+" "+update.Message.Chat.LastName+"\n Please enter the district name and press send")
			msg.ReplyToMessageID = update.Message.MessageID
			bot.Send(msg)
		}

		redisClient, err := cache.CreateConnection("localhost", "", 6379)
		if err != nil {
			fmt.Errorf("%q", err)
		}

		rh, err := cache.NewReJSONHandler(redisClient)
		if err != nil {
			fmt.Errorf("%q", err)
		}

		if update.Message.Text != "/start" {
			district, err := districtExists(rh, strings.ToLower(update.Message.Text))
			if err != nil {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, strings.ToUpper(update.Message.Text)+" is not a Valid District Name")
				msg.ReplyToMessageID = update.Message.MessageID
				bot.Send(msg)
			} else {
				records, err := redis.Bytes(rh.Get("cowin", "."))
				if err != nil {
					fmt.Errorf("%q", err)
				}

				var resultData []HospitalsData
				err = json.Unmarshal(records, &resultData)
				if err != nil {
					fmt.Println(err)
				}

				var finalData []string

				for _, value := range resultData {
					if strings.ToLower(value.DistrictName) == strings.ToLower(update.Message.Text) {
						for _, hospital := range value.Hospital {
							if hospital.Date == time.Now().Format("02-01-2006") || hospital.Date == time.Now().Add(1440*time.Minute).Format("02-01-2006") || hospital.Date == time.Now().Add(2880*time.Minute).Format("02-01-2006") {
								data := fmt.Sprintf("   Hospital: %s\n   MinAgeLimit: %d\n   Date: %s\n   AvailableCapacity:%d\n   Vaccine: %s\n   District: %s\n\n\n", hospital.Name, int(hospital.MinAgeLimit), hospital.Date, int(hospital.AvailableCapacity), hospital.Vaccine, hospital.District)
								if len(finalData) < 10 {
									finalData = append(finalData, data)
								}
							}
						}
					}
				}

				fmt.Println("Number of hospitals where vaccine is available for the city: ", update.Message.Text, " is: ", len(finalData))

				if len(finalData) == 0 {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Hi "+update.Message.Chat.FirstName+" "+update.Message.Chat.LastName+"\nNo slots available for the city: "+district+" for the next 2 days")
					msg.ReplyToMessageID = update.Message.MessageID
					bot.Send(msg)
				}

				for len(finalData) > 0 {
					if len(finalData) <= 5 {
						stringData := strings.Join(finalData, "\n")
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, stringData)
						msg.ReplyToMessageID = update.Message.MessageID
						bot.Send(msg)
						finalData = finalData[len(finalData):]
					} else {
						stringData := strings.Join(finalData[:5], "\n")
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, stringData)
						msg.ReplyToMessageID = update.Message.MessageID
						bot.Send(msg)
						finalData = finalData[5:]
					}
				}
			}
		}
	}
}
