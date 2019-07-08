package insidescraper

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"path"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	scraper := InsideScraper{}

	if err := scraper.Scrape(); err != nil {
		fmt.Println("Error in scrape: " + err.Error())
	}

	output, _ := json.Marshal(scraper.Site())
	fmt.Println("Site data:\n\n", string(output))
}

func TestValidateJSON(t *testing.T) {
	jsonText, _ := ioutil.ReadFile("scraped.json")
	var site []SiteSection
	json.Unmarshal(jsonText, &site)

	siteMap := make(map[string]SiteSection, len(site))
	for _, section := range site {
		siteMap[section.ID] = section
	}

	fmt.Print("Sections which were not loaded\n")
	for key, section := range siteMap {
		for _, subSectionID := range section.Sections {
			_, exists := siteMap[subSectionID]
			if !exists {
				fmt.Println(subSectionID, "\n(", key, ")")
				printPossibleMatches(siteMap, subSectionID)
				fmt.Print("\n")
			}
		}
	}

	fmt.Print("\n\nSections with no content\n")

	for key, section := range siteMap {
		for _, subSectionID := range section.Sections {
			if subSection, exists := siteMap[subSectionID]; exists {
				if len(subSection.Sections) == 0 && len(subSection.Lessons) == 0 {
					fmt.Println(subSectionID, "\n(", key, ")")
					printPossibleMatches(siteMap, subSectionID)
					fmt.Print("\n")
				}
			}
		}
	}
}

func printPossibleMatches(site map[string]SiteSection, id string) {
	possibleMatches := getPossibleMatches(site, id)

	for _, match := range possibleMatches {
		fmt.Println("maybe -> ", match)

	}
}

func getPossibleMatches(site map[string]SiteSection, id string) []string {
	matches := make([]string, 0, 10)
	idBase := path.Base(id)

	for key := range site {
		if key == id {
			continue
		}

		keyBase := path.Base(key)
		if keyBase == idBase {
			matches = append(matches, key)
		} else if strings.Contains(keyBase, idBase) || strings.Contains(idBase, keyBase) && math.Abs(float64(len(keyBase))-float64(len(idBase))) < 6 {
			matches = append(matches, key)
		}
	}

	return matches
}
