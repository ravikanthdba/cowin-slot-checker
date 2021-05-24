package scheduler

import (
	"github.com/jinzhu/gorm"
)

type Postoffice struct {
	gorm.Model
	Name     string `json:"Name"`
	Circle   string `json:"Circle"`
	District string `json:"District"`
	Division string `json:"Division"`
	Region   string `json:"Region"`
	Block    string `json:"Block"`
	State    string `json:"State"`
	Pincode  string `json:"Pincode"`
}

type Hospital struct {
	gorm.Model
	StateName              string
	District          	   string
	Name              	   string
	BlockName        	   string
	Date              	   string
	AvailableCapacity 	   float64
	MinAgeLimit       	   float64
	Vaccine           	   string
	Pincode 		  	   float64
	AvailableCapacityDose1 float64
	AvailableCapacityDose2 float64
}