package main

import (
	"encoding/json"
	"fmt"
)

func main() {
	scraper := InsideScraper{}
	
	if err := scraper.Scrape(); err != nil {
		fmt.Println("Error in scrape: " + err.Error())
	}

	output, _ := json.Marshal(scraper.Site())
	fmt.Println("Site data:\n\n", string(output))
}
