package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/spf13/viper"
)

// Define the JSON response structure of the response
type Meta struct {
	AutoAdded           bool   `json:"auto_added"`
	ManagedByApps       bool   `json:"managed_by_apps"`
	ManagedByArgoTunnel bool   `json:"managed_by_argo_tunnel"`
	Source              string `json:"source"`
}

type Result struct {
	ID        string `json:"id"`
	ZoneID    string `json:"zone_id"`
	ZoneName  string `json:"zone_name"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Content   string `json:"content"`
	Proxiable bool   `json:"proxiable"`
	Proxied   bool   `json:"proxied"`
	TTL       int    `json:"ttl"`
	Locked    bool   `json:"locked"`
	Meta      Meta   `json:"meta"`
}

type ResultInfo struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Count      int `json:"count"`
	TotalCount int `json:"total_count"`
	TotalPages int `json:"total_pages"`
}

// APIResponse is used when an array of results are returned for a request
type APIResponse struct {
	Result     []Result      `json:"result"`
	Success    bool          `json:"success"`
	Errors     []interface{} `json:"errors"`
	Messages   []interface{} `json:"messages"`
	ResultInfo ResultInfo    `json:"result_info"`
}

// UpdateResponse is used when response contains only one result, instead of an array
type UpdateResponse struct {
	Result     Result        `json:"result"`
	Success    bool          `json:"success"`
	Errors     []interface{} `json:"errors"`
	Messages   []interface{} `json:"messages"`
	ResultInfo ResultInfo    `json:"result_info"`
}

type Config struct {
	AuthEmail      string `mapstructure:"AUTH_EMAIL"`
	AuthMethod     string `mapstructure:"AUTH_METHOD"`
	AuthKey        string `mapstructure:"AUTH_KEY"`
	ZoneIdentifier string `mapstructure:"ZONE_IDENTIFIER"`
	RecordName     string `mapstructure:"RECORD_NAME"`
	Ttl            string `mapstructure:"TTL"`
	Proxy          bool   `mapstructure:"PROXY"`
	Sitename       string `mapstructure:"SITENAME"`
	Verbose        bool   `mapstructure:"VERBOSE"`
}

// LoadConfig maps the config added in app.env file and unmarshals all the values a Config structure
func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	err = viper.Unmarshal(&config)
	return config, err
}

func main() {

	config, err := LoadConfig(".")
	if err != nil {
		log.Fatal("cannot load config:", err)
	}

	// Try getting our Public IP
	log.Println("Attempting to get our current IP address from ipify.org")
	ip, err := http.Get("https://api.ipify.org?format=text")
	if err != nil || ip.StatusCode != http.StatusOK {
		log.Fatalf("Failed to fetch IP Address from ipify! \nStatusCode: %v || err: %v\n", ip.StatusCode, err)
		os.Exit(1)
	}
	defer ip.Body.Close()

	bodyBytes, err := io.ReadAll(ip.Body)
	if err != nil {
		log.Fatal(err)
	}
	currentIP := string(bodyBytes)
	log.Printf("Our Current Public IP Address: %v", currentIP)

	// Check and set the proper auth header based on auth method
	var auth_header string
	var ifBearer string
	if config.Verbose {
		fmt.Printf("Auth Method: %s\n", config.AuthMethod)
	}
	if config.AuthMethod == "global" {
		auth_header = "X-Auth-Key"
		ifBearer = ""

	} else if config.AuthMethod == "token" {
		auth_header = "Authorization"
		ifBearer = "Bearer "
	} else {
		log.Fatalf("Failed to set Auth Header | Provided Auth Method: %v\n", config.AuthMethod)
		os.Exit(1)
	}

	// Fetch the current IP address set in Cloudflare
	log.Println("Checking for A record")
	endpoint := fmt.Sprintf(
		"https://api.cloudflare.com/client/v4/zones/%s/dns_records?type=A&name=%s",
		config.ZoneIdentifier, config.RecordName,
	)
	client := &http.Client{}
	request, _ := http.NewRequest("GET", endpoint, nil)
	request.Header.Set("X-Auth-Email", config.AuthEmail)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(auth_header, ifBearer+config.AuthKey)

	response, err := client.Do(request)
	if err != nil {
		log.Fatalf("Failed to fetch records from cloudflare. \nError: %v\n", err)
		os.Exit(1)
	}
	defer response.Body.Close()

	recordBytes, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatalf("Something went wrong. Failed to read records. \nError: %v\n", err)
		os.Exit(1)
	}

	// Unmarshal response to APIResonse object
	var records APIResponse
	err = json.Unmarshal(recordBytes, &records)
	if err != nil {
		log.Fatalf("Failed to unmarshal JSON records. \nError: %v\n", err)
		os.Exit(1)
	}

	// Check if the domain has an A record
	fmt.Printf("Count of Records: %v\n", records.ResultInfo.Count)
	if records.ResultInfo.Count < 1 {
		log.Fatalf("Records does not exist for %v. Try adding one first.", config.Sitename)
		os.Exit(1)
	}

	// If Verbose, pretty print the response
	if config.Verbose {
		prettyJSON, err := json.MarshalIndent(records, "", "  ")
		if err != nil {
			log.Fatalf("Failed to pretty print response. \n Error: %v", err)
		}
		fmt.Println(string(prettyJSON))
	}

	log.Print("The number of records seems good. Moving on")

	// Extract existing A record for site from response
	oldIP := records.Result[0].Content
	if oldIP == currentIP {
		log.Println("Your public IP hasn't changed since last update. I will try again later")
		os.Exit(0)
	}

	// Extract record identifier from response
	recordIdentifier := records.Result[0].ID

	// Change the A record in Cloudflare using the API
	updateUrl := fmt.Sprintf(
		"https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s",
		config.ZoneIdentifier, recordIdentifier,
	)
	payload := map[string]any{
		"type":    "A",
		"name":    config.RecordName,
		"content": currentIP,
		"ttl":     config.Ttl,
		"proxied": config.Proxy,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Fatalf("Failed to encode JSON data to update records: %v\n", err)
		os.Exit(1)
	}

	request, _ = http.NewRequest("PATCH", updateUrl, bytes.NewBuffer(payloadBytes))
	request.Header.Set("X-Auth-Email", config.AuthEmail)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(auth_header, ifBearer+config.AuthKey)

	response, err = client.Do(request)
	if err != nil {
		log.Printf("Failed to update records in Cloudflare: %v\n", err)
	}
	defer response.Body.Close()

	var updateResponse UpdateResponse
	recordBytes, _ = io.ReadAll(response.Body)
	err = json.Unmarshal(recordBytes, &updateResponse)
	if err != nil {
		log.Fatalf("Failed to unmarshal JSON records: %v", err)
		os.Exit(1)
	}
	if config.Verbose {
		prettyJSON, err := json.MarshalIndent(updateResponse, "", "  ")
		if err != nil {
			log.Fatalf("Failed to pretty print response. \n Error: %v", err)
		}
		fmt.Printf("New Records: %v\n", string(prettyJSON))
	}

	switch true {
	case updateResponse.Success:
		log.Printf(
			"Successfully updated records for %v (%v): %v\n",
			recordIdentifier, config.RecordName, updateResponse.Result.Content,
		)
	case !updateResponse.Success:
		log.Fatalf("Failed to update records for %v (%v)\n", recordIdentifier, config.RecordName)

	default:
		return
	}
}
