package main

import (
	// flags
	"bytes"
	"encoding/json"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
)

type Config struct {
	APIKey      string
	Output      string
	InputDomain string
	Mode        string
	SearchType  string
}

var config *Config

func LoadConfig() *Config {
	config = &Config{}
	flag.StringVar(&config.APIKey, "token", "", "API token")
	flag.StringVar(&config.Output, "output", "", "Output file")
	flag.StringVar(&config.InputDomain, "domain", "", "Input domain")
	flag.StringVar(&config.Mode, "mode", "purchase", "Mode (purchase, or preview)")
	flag.StringVar(&config.SearchType, "searchtype", "current", "Search type (current, or historic)")

	flag.Parse()

	apiToken := os.Getenv("WHOIS_API_KEY")

	if config.APIKey == "" {
		if apiToken != "" {
			config.APIKey = apiToken
		}
	}

	if config.Output == "" {
		config.Output = "/dev/stdout"
	}

	return config
}

func ExitWithError(err string) {
	log.Printf("[!] %s", err)
	log.Println("Exiting...")

	os.Exit(1)
}

func GetNameservers(domain string) []string {
	ns, err := net.LookupNS(domain)
	if err != nil {
		ExitWithError("Failed to get nameservers for domain.")
	}

	var nameservers []string
	for _, server := range ns {
		if strings.Contains(server.Host, "awsdns") {
			log.Printf("Skipping AWS nameserver: %s\n", server.Host)
			continue
		}
		if strings.Contains(server.Host, "nsone") {
			log.Printf("Skipping NS1 nameserver: %s\n", server.Host)
			continue
		}
		if strings.Contains(server.Host, "ns.cloudflare") {
			log.Printf("Cloudflare nameservers detected! Make sure to review the output since Cloudflare nameservers are shared by many domains at the same time.\n")
		}
		nameservers = append(nameservers, server.Host)
	}

	return nameservers
}

func GetWhoisData(domain string) []string {
	// curl  https://reverse-whois.whoisxmlapi.com/api/v2 --data '{"apikey":"'$WHOIS_API_KEY'","searchtype":"current","mode":"purchase","advancedSearchTerms":[{"field":"NameServers","term":"'"pdns210.ultradns.org."'"},{"field":"NameServers","term":"'"pdns210.ultradns.net."'"}]}' | jq .domainsList
	url := "https://reverse-whois.whoisxmlapi.com/api/v2"
	requestBody := struct {
		APIKey              string `json:"apikey"`
		SearchType          string `json:"searchtype"`
		Mode                string `json:"mode"`
		AdvancedSearchTerms []struct {
			Field string `json:"field"`
			Term  string `json:"term"`
		} `json:"advancedSearchTerms"`
	}{}
	requestBody.APIKey = config.APIKey
	requestBody.SearchType = config.SearchType
	requestBody.Mode = config.Mode

	NameServers := GetNameservers(domain)
	addedNameservers := 0
	for _, ns := range NameServers {
		if addedNameservers >= 4 {
			break
		}
		requestBody.AdvancedSearchTerms = append(requestBody.AdvancedSearchTerms, struct {
			Field string `json:"field"`
			Term  string `json:"term"`
		}{Field: "NameServers", Term: ns})
		addedNameservers++
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		ExitWithError(err.Error())
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		ExitWithError(err.Error())
	}

	defer resp.Body.Close()

	// jq .domainsList[]
	response := struct {
		DomainsList  []string `json:"domainsList"`
		DomainsCount int      `json:"domainsCount"`
	}{}
	body := new(bytes.Buffer)
	body.ReadFrom(resp.Body)

	log.Println(body.String())

	bodyString := body.String()
	err = json.Unmarshal([]byte(bodyString), &response)
	if err != nil {
		ExitWithError(err.Error())
	}

	if config.Mode == "preview" {
		log.Printf("Found %d domains for nameservers: %v\n", response.DomainsCount, NameServers)
		return []string{}
	}

	if len(response.DomainsList) == 0 {
		log.Printf("No domains found for nameservers: %v\n", NameServers)
		return []string{}
	}

	log.Printf("Found %d domains for nameservers: %v\n", len(response.DomainsList), NameServers)
	return response.DomainsList

}

func main() {
	config := LoadConfig()
	if config.APIKey == "" {
		ExitWithError("API token is required. Please provide it using the -token flag or WHOIS_API_KEY environment variable.")
	}
	if config.InputDomain == "" {
		ExitWithError("Input domain is required. Please provide it using the -domain flag.")
	}

	whoisData := GetWhoisData(config.InputDomain)
	for _, domain := range whoisData {
		// write to output file
		f, err := os.OpenFile(config.Output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			ExitWithError(err.Error())
		}
		defer f.Close()

		_, err = f.WriteString(domain + "\n")
		if err != nil {
			ExitWithError(err.Error())
		}
	}

}
