package insidescraper

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

func TestRun(t *testing.T) {
	runScraper()
}

func TestValidateJSON(t *testing.T) {
	site := getSite()
	postScraper := PostScraper{
		Site: site.Sections,
	}

	fmt.Print("Sections which were not loaded\n")
	printCorrections(postScraper.GetMissingCorrections())

	fmt.Print("\n\nSections with no content\n")
	printCorrections(postScraper.GetEmptyCorrections())

	fmt.Print("\nMissing lessons")
	for _, section := range site.Sections {
		for _, lessonID := range section.Lessons {
			if _, exists := site.Lessons[lessonID]; !exists {
				fmt.Print(section.ID + ": missing: " + lessonID)
			}
		}
	}
}

func TestApplyFix(t *testing.T) {
	postScraper := PostScraper{
		Site: getSite().Sections,
	}

	postScraper.FixSite()

	file, _ := os.Create("auto_fixed.json")
	defer file.Close()

	jsonOut, _ := json.Marshal(postScraper.Site)
	file.Write(jsonOut)
}

func TestGetNotFixed(t *testing.T) {
	postScraper := PostScraper{
		Site: getSite().Sections,
	}

	postScraper.FixSite()

	fmt.Println("Broken: Missing")
	printNotFixed(postScraper.Missing)
	fmt.Println("\n\nBroken: Empty")
	printNotFixed(postScraper.Empty)
}

func runScraper(scraperURL ...string) {
	scraper := InsideScraper{}

	if err := scraper.Scrape(scraperURL...); err != nil {
		fmt.Println("Error in scrape: " + err.Error())
	}

	output, _ := json.MarshalIndent(scraper.Site, "", "    ")
	fmt.Println("Site data:\n\n", string(output))
}

func printNotFixed(corrections map[string]Correction) {
	for badID, correction := range corrections {
		if correction.WasCorrected {
			continue
		}

		fmt.Println(badID)

		for _, parent := range correction.Parents {
			fmt.Println("(" + parent + ")")
		}

		if correction.Is404 {
			fmt.Println("404")
		}

		fmt.Println()
	}
}

func getSite() Site {
	jsonText, _ := ioutil.ReadFile("scraped.json")
	var site Site
	json.Unmarshal(jsonText, &site)

	return site
}

func printCorrections(corrections map[string]Correction) {
	for key, correction := range corrections {
		fmt.Println(key)

		if correction.Is404 {
			fmt.Println("404")
		}

		if correction.IsConfirmed {
			fmt.Print("CONFIRMED")
		}

		if len(correction.Guesses) > 0 {
			fmt.Println("-> " + correction.Guesses[0])
		}
		fmt.Print("\n")
	}
}
