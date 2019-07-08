package insidescraper

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
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
	if response, err := http.Head(id); err == nil {
		if response.StatusCode == http.StatusNotFound {
			fmt.Println("404")
		}
	}

	matches := getPossibleMatches(site, id)

	if matches != "" {
		startText := "maybe -> "

		body1 := getBody(id)
		body2 := getBody(matches)

		if body1 != "" && body1 == body2 {
			startText = "CONFIRMED MATCH: "
		}

		fmt.Println(startText, matches)
	}
}

func getBody(url string) string {
	response, err := http.Get(url)

	if err != nil {
		return ""
	}

	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)

	if err != nil {
		return ""
	}

	return string(body)
}

func getPossibleMatches(site map[string]SiteSection, id string) string {
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

	if len(matches) > 0 {
		return matches[0]
	}

	return ""
}
