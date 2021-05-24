package cowin

type PostalCodeResponse struct {
	Postoffice []Postoffice `json:"PostOffice"`
}

type Postoffice struct {
	Name           string      `json:"Name"`
	Circle         string      `json:"Circle"`
	District       string      `json:"District"`
	Division       string      `json:"Division"`
	Region         string      `json:"Region"`
	Block          string      `json:"Block"`
	State          string      `json:"State"`
	Pincode        string      `json:"Pincode"`
}

type CowinStatesResponse struct {
	States []struct {
		StateID   int    `json:"state_id"`
		StateName string `json:"state_name"`
	} `json:"states"`
}

type CowinDistrictsResponse struct {
	Districts []struct {
		DistrictID   int    `json:"district_id"`
		DistrictName string `json:"district_name"`
	} `json:"districts"`
}

type CowinHospitalsResponse struct {
	Centers []Centers `json:"centers"`
}

type Centers struct {
	CenterID     int        `json:"center_id"`
	Name         string     `json:"name"`
	Address      string     `json:"address"`
	StateName    string     `json:"state_name"`
	DistrictName string     `json:"district_name"`
	BlockName    string     `json:"block_name"`
	Pincode      float64        `json:"pincode"`
	Lat          float64        `json:"lat"`
	Long         float64        `json:"long"`
	From         string     `json:"from"`
	To           string     `json:"to"`
	FeeType      string     `json:"fee_type"`
	Sessions     []Sessions `json:"sessions"`
}

type Sessions struct {
	Date                   string   `json:"date"`
	AvailableCapacity      float64      `json:"available_capacity"`
	MinAgeLimit            float64      `json:"min_age_limit"`
	Vaccine                string   `json:"vaccine"`
	Slots                  []string `json:"slots"`
	AvailableCapacityDose1 float64      `json:"available_capacity_dose1"`
	AvailableCapacityDose2 float64      `json:"available_capacity_dose2"`
}
