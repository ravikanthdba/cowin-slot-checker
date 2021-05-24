package scheduler

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gammazero/workerpool"

	database "github.com/cowin-slot-checker/cowin-slot-checker-worker/src/database"
	model "github.com/cowin-slot-checker/cowin-slot-checker-worker/src/model"
)

func getPinCodes() ([]string, error) {
	log.Println("Fetching the pincodes based on hospitals data")
	start := time.Now()
	var pincodes []string

	db, err := database.CreateConnection("cowin_user", "GenieHello^123", "172.23.227.40", "cowin_database")
	if err != nil {
		fmt.Errorf("%q", err)
	}

	query, err := db.DatabaseQueryWithoutCondition("select distinct pincode from hospitals order by pincode")
	if err != nil {
		fmt.Errorf("%q", err)
	}

	records, err := query.Rows()
	if err != nil {
		fmt.Errorf("%q", err)
	}

	for records.Next() {
		var pincode string
		if err := records.Scan(&pincode); err != nil {
			fmt.Errorf("%q", err)
		}

		pincodes = append(pincodes, pincode)
	}

	fmt.Println("Pincodes are: ", pincodes)
	log.Println("Fetcing the Pincodes done in: ", time.Since(start))
	return pincodes, nil
}

type PostalCodes struct {
	postalCodes []model.PostalCodeResponse
}

func (postcode PostalCodes) getPostalCodesData() ([]Postoffice, error) {
	if len(postcode.postalCodes) == 0 {
		return nil, fmt.Errorf("%q", "input dataset for postcodes is empty")
	}

	var postalcodes []Postoffice
	for _, value := range postcode.postalCodes {
		for _, code := range value.Postoffice {
			var p Postoffice
			p.Name = code.Name
			p.District = code.District
			p.Block = code.Block
			p.State = code.State
			p.Pincode = code.Pincode
			postalcodes = append(postalcodes, p)
		}
	}
	return postalcodes, nil
}

type PostOffices struct {
	postOffice [][]Postoffice
}

var pdata PostOffices

func CapturePostalCodesTask() {
	log.Println("Launching Background Process for Postal Code Data")
	start := time.Now()
	postalCodeClient, err := model.NewPostalCodeClient()
	if err != nil {
		fmt.Errorf("%q", err)
	}

	pincodes, err := getPinCodes()
	if err != nil {
		fmt.Errorf("%q", err)
	}

	wp := workerpool.New(40)

	for _, code := range pincodes {
		urlValue, err := getParsedURL("https://api.postalpincode.in/pincode/" + code)
		if err != nil {
			fmt.Errorf("%q", err)
		}

		req, err := postalCodeClient.NewRequest(http.MethodGet, urlValue, nil)
		if err != nil {
			fmt.Errorf("%q", err)
		}

		wp.Submit(func() {
			log.Println("Launching for code: ", code)
			var postCode PostalCodes
			postCode.postalCodes, err = postalCodeClient.GetPostalCodes(req, code)
			if err != nil {
				fmt.Errorf("%q", err)
			}

			data, err := postCode.getPostalCodesData()
			if err != nil {
				fmt.Errorf("%q", err)
			}

			mu.Lock()
			if len(data) > 0 {
				pdata.postOffice = append(pdata.postOffice, data)
			}
			mu.Unlock()
		})
	}

	wp.StopWait()
	log.Println("Refresh of Postal Codes completed")
	log.Println("Function to refresh Postal Codes sleeping for 6 months")
	log.Println("Background process to capture Postal Code data completed in: ", time.Since(start))
}
