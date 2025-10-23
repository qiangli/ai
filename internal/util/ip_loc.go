package util

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const apiURL = "https://ipapi.co/json/"

type LocationData struct {
	IP          string  `json:"ip"`
	Country     string  `json:"country_name"`
	CountryCode string  `json:"country_code"`
	Region      string  `json:"region"`
	City        string  `json:"city"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	TimeZone    string  `json:"timezone"`
	Org         string  `json:"org"`
	ASN         string  `json:"asn"`
}

// fetchLocation makes an API request to https://ipapi.co and returns location data.
func FetchLocation() (string, error) {
	resp, err := http.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch location data: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("received non-200 response status: %d", resp.StatusCode)
	}

	var data LocationData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", fmt.Errorf("failed to decode location data: %v", err)
	}

	return displayLocation(&data), nil
}

// displayLocation prints the location data in a Markdown table format.
func displayLocation(data *LocationData) string {
	var result string
	result += fmt.Sprintln("| Field         | Value              |")
	result += fmt.Sprintln("|---------------|--------------------|")
	result += fmt.Sprintf("| IP            | %s |\n", data.IP)
	result += fmt.Sprintf("| Country       | %s |\n", data.Country)
	result += fmt.Sprintf("| Country Code  | %s |\n", data.CountryCode)
	result += fmt.Sprintf("| Region        | %s |\n", data.Region)
	result += fmt.Sprintf("| City          | %s |\n", data.City)
	result += fmt.Sprintf("| Latitude      | %.4f |\n", data.Latitude)
	result += fmt.Sprintf("| Longitude     | %.4f |\n", data.Longitude)
	result += fmt.Sprintf("| Timezone      | %s |\n", data.TimeZone)
	result += fmt.Sprintf("| ISP           | %s |\n", data.Org)
	result += fmt.Sprintf("| ASN           | %s |\n", data.ASN)
	return result
}
