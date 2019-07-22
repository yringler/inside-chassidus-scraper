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
	site := getSite()
	postScraper := PostScraper{
		Site: site.Sections,
	}

	postScraper.FixSite()

	file, _ := os.Create("auto_fixed.json")
	defer file.Close()

	jsonOut, _ := json.MarshalIndent(site, "", "    ")
	file.Write(jsonOut)
}

// Check how many issues are in the data post fix.
// This is only relevant now that sections can be converted to lessons.
func TestAppliedFix(t *testing.T) {
	site := getSite("auto_fixed.json")

	// Print every section which has no lessons or sections, and check that it's lessons and sections exist.
	fmt.Print("Searching for bad data in sections.\n\n")
	for sectionID, section := range site.Sections {
		if len(section.Lessons) == 0 && len(section.Sections) == 0 {
			fmt.Println(sectionID + ": contains no content")
		}

		for _, childSection := range section.Sections {
			if _, exists := site.Sections[childSection]; !exists {
				fmt.Println(sectionID + ":\nContains missing section:" + childSection)
			}
		}

		for _, lesson := range section.Lessons {
			if _, exists := site.Lessons[lesson]; !exists {
				fmt.Println(sectionID + ":\nContains missing lesson:" + lesson)
			}
		}
	}

	// Print every lesson which has no media
	fmt.Print("Searching for bad data in lessons.\n\n")
	for lessonID, lesson := range site.Lessons {
		if len(lesson.Audio) == 0 {
			fmt.Println(lessonID + ": contains no audio")
		}
	}
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

func getSite(jsonPath ...string) Site {
	jsonFile := "scraped.json"
	if len(jsonPath) != 0 {
		jsonFile = jsonPath[0]
	}
	jsonText, _ := ioutil.ReadFile(jsonFile)
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
