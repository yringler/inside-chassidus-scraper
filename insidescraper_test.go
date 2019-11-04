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
	postScraper := PostScraper{
		Site: getSite(),
	}

	fmt.Print("Sections which were not loaded\n")
	printCorrections(postScraper.GetMissingCorrections())

	fmt.Print("\n\nSections with no content\n")
	printCorrections(postScraper.GetEmptyCorrections())

	fmt.Print("\nMissing lessons")
	for _, section := range postScraper.Site.Sections {
		for _, lessonID := range section.Lessons {
			if _, exists := postScraper.Site.Lessons[lessonID]; !exists {
				fmt.Print(section.ID + ": missing: " + lessonID)
			}
		}
	}
}

func TestApplyFix(t *testing.T) {
	postScraper := PostScraper{
		Site: getSite(),
	}

	postScraper.FixSite()

	file, _ := os.Create("auto_fixed.json")
	defer file.Close()

	jsonOut, _ := json.MarshalIndent(postScraper.Site, "", "    ")
	file.Write(jsonOut)
}

// Check how many issues are in the data post fix.
// This is only relevant now that sections can be converted to lessons.
func TestAppliedFix(t *testing.T) {
	site := getSite("auto_fixed.json")

	badSections := make(map[string]string, 0)
	badLessons := make(map[string]string, 0)

	// Print every section which has no lessons or sections, and check that it's lessons and sections exist.
	fmt.Print("Searching for bad data in sections.\n\n")
	for sectionID, section := range site.Sections {
		if len(section.Lessons) == 0 && len(section.Sections) == 0 {
			if _, exists := badSections[sectionID]; !exists {
				fmt.Println(sectionID + ": contains no content")
				badLessons[sectionID] = sectionID
			}
		}

		for _, childSection := range section.Sections {
			if _, exists := site.Sections[childSection]; !exists {
				if _, exists := badSections[childSection]; !exists {
					fmt.Println(sectionID + ":\nContains missing section:" + childSection)
					badSections[childSection] = childSection
				}
			}
		}

		for _, lesson := range section.Lessons {
			if _, exists := site.Lessons[lesson]; !exists {
				if _, exists := badLessons[lesson]; !exists {
					fmt.Println(sectionID + ":\nContains missing lesson:" + lesson)
					badLessons[lesson] = lesson
				}
			}
		}
	}

	// Print every lesson which has no media
	fmt.Print("Searching for bad data in lessons.\n\n")
	for lessonID, lesson := range site.Lessons {
		if len(lesson.Audio) == 0 && len(lesson.Pdf) == 0 {
			fmt.Println(lessonID + ": contains no audio or PDF")
		}
	}
}

func TestGetNotFixed(t *testing.T) {
	postScraper := PostScraper{
		Site: getSite(),
	}

	postScraper.FixSite()

	fmt.Println("Broken: Missing")
	printNotFixed(postScraper.Missing)
	fmt.Println("\n\nBroken: Empty")
	printNotFixed(postScraper.Empty)
}

func TestFullRun(t *testing.T) {
	scraper := InsideScraper{}
	scraper.Scrape("https://insidechassidus.org/")

	postScraper := PostScraper{
		Site: scraper.Site,
	}
	postScraper.FixSite()

	counter := MakeCounter(&postScraper.Site)
	counter.CountLessons()

	file, _ := os.Create("full_run_data.json")
	defer file.Close()

	jsonOut, _ := json.MarshalIndent(postScraper.Site, "", "\t")
	file.Write(jsonOut)
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
	jsonFile := "scraped.2.json"
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
