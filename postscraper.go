package insidescraper

import (
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"path"
	"reflect"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// DataType is the type of the site data.
type DataType int

const (
	// SectionType refers to a section.
	SectionType DataType = iota
	// LessonType refers to a lesson.
	LessonType
)

// PostScraper goes over the scraped data and fixes it up as much as possible.
type PostScraper struct {
	Site    Site
	Missing map[string]Correction
	Empty   map[string]Correction
}

// Correction is a (possible) correction for a missing link.
type Correction struct {
	Guesses []string
	// The sections which reference the bad URL.
	Parents      []string
	Is404        bool
	IsConfirmed  bool
	WasCorrected bool
	Source       DataType
}

// FixSite applies fixes to the site data.
func (cleaner *PostScraper) FixSite() {
	cleaner.Missing = cleaner.GetMissingCorrections()
	cleaner.Empty = cleaner.GetEmptyCorrections()

	for badID, correction := range cleaner.Missing {
		cleaner.applyFix(badID, &correction)
		cleaner.Missing[badID] = correction
	}

	for badID, correction := range cleaner.Empty {
		cleaner.applyFix(badID, &correction)
		cleaner.Empty[badID] = correction
	}
}

// GetMissingCorrections attempts to find as many missing sections as possible.
func (cleaner *PostScraper) GetMissingCorrections() map[string]Correction {
	corrections := make(map[string]Correction, 10)

	for parentID, section := range cleaner.Site.Sections {
		for _, subSectionID := range section.Sections {
			// Don't find correction twice.
			if _, exists := corrections[subSectionID]; exists {
				addParent(corrections, subSectionID, parentID)
			} else if _, exists := cleaner.Site.Sections[subSectionID]; !exists {
				corrections[subSectionID] = cleaner.getPossibleMatches(subSectionID, parentID)
				// No need to test for lessons here. If this sectionID really references a lesson,
				// it would already have been converted in the scraper.
				// And if it incorrectly references a lesson, that will get picked up in getPossibleMatches, also.
			}
		}
	}

	return corrections
}

// addParent registers the given section as referencing the given bad section/lesson ID.
func addParent(corrections map[string]Correction, badID, parentID string) {
	correction, _ := corrections[badID]
	correction.Parents = append(correction.Parents, parentID)
	corrections[badID] = correction
}

// GetEmptyCorrections finds all empty sections (no lessons or subsections) and tries to correct.
func (cleaner *PostScraper) GetEmptyCorrections() map[string]Correction {
	corrections := make(map[string]Correction, 10)

	for parentID, section := range cleaner.Site.Sections {
		for _, subSectionID := range section.Sections {
			// Don't try to correct the same thing twice.
			if _, exists := corrections[subSectionID]; exists {
				addParent(corrections, subSectionID, parentID)
			} else if subSection, exists := cleaner.Site.Sections[subSectionID]; exists {
				if len(subSection.Sections) == 0 && len(subSection.Lessons) == 0 {
					corrections[subSectionID] = cleaner.getPossibleMatches(subSectionID, parentID)
				}
			}
		}

		for _, lessonID := range section.Lessons {
			// Don't try to correct the same thing twice.
			if _, exists := corrections[lessonID]; exists {
				addParent(corrections, lessonID, parentID)
			} else if lesson, exists := cleaner.Site.Lessons[lessonID]; exists {
				if len(lesson.Pdf) == 0 && len(lesson.Audio) == 0 {
					corrections[lessonID] = cleaner.getPossibleMatches(lessonID, parentID)
				}
			}
		}
	}

	return corrections
}

// applyFix fixes up the site based on the correction. If the correction is executed, marked as such.
func (cleaner *PostScraper) applyFix(badID string, correction *Correction) {
	if len(correction.Guesses) > 0 && (correction.IsConfirmed || correction.Is404) {
		if _, exists := cleaner.Site.Sections[badID]; exists {
			delete(cleaner.Site.Sections, badID)
		}

		if _, exists := cleaner.Site.Lessons[badID]; exists {
			delete(cleaner.Site.Lessons, badID)
		}

		for sectionID, section := range cleaner.Site.Sections {
			// Keep track of the good sections.
			// "sectionIDs" which actually reference lessons aren't added.
			goodSections := make([]string, 0, len(section.Sections))

			for _, subSectionID := range section.Sections {
				if subSectionID == badID {
					if correction.Source == SectionType {
						goodSections = append(goodSections, correction.Guesses[0])
					} else {
						section.Lessons = append(section.Lessons, correction.Guesses[0])
					}
				} else {
					goodSections = append(goodSections, subSectionID)
				}
			}

			// Work on lessons which were converted from sections.
			for i, lessonID := range section.Lessons {
				if lessonID == badID && correction.Source == LessonType {
					section.Lessons[i] = correction.Guesses[0]
				}
			}

			section.Sections = goodSections
			cleaner.Site.Sections[sectionID] = section
		}

		correction.WasCorrected = true
	}
}

// Create corrections for a parents reference to a bad ID.
func (cleaner *PostScraper) getPossibleMatches(id, parentID string) Correction {
	correction := Correction{
		Parents: []string{parentID},
	}

	if response, err := http.Head(id); err == nil {
		if response.StatusCode == http.StatusNotFound {
			correction.Is404 = true
		}
	}

	correction.Guesses, correction.Source = cleaner.getPossibleIdsFromSite(id)

	if correction.Guesses != nil {
		doc1, err := goquery.NewDocument(id)
		if err != nil {
			fmt.Println("Error in get pos: ", err)
			return correction
		}
		doc2, err := goquery.NewDocument(correction.Guesses[0])
		if err != nil {
			fmt.Println("Error in get pos: ", err)
			return correction
		}

		content1 := doc1.Find("#main_container")
		content2 := doc2.Find("#main_container")

		html1, _ := content1.Html()
		html2, _ := content2.Html()

		if html1 != "" && html1 == html2 {
			correction.IsConfirmed = true
		}
	}

	return correction
}

// Searches lessons and sections for matching IDs.
func (cleaner *PostScraper) getPossibleIdsFromSite(id string) ([]string, DataType) {

	if matches := getPossibleIDFromSiteDataType(cleaner.Site.Sections, id); matches != nil {
		return matches, SectionType
	}

	if matches := getPossibleIDFromSiteDataType(cleaner.Site.Lessons, id); matches != nil {
		return matches, LessonType
	}

	return nil, 0
}

// Searches one data type (either lessons or sections) for matching IDs.
func getPossibleIDFromSiteDataType(data interface{}, badID string) (matches []string) {
	for _, key := range reflect.ValueOf(data).MapKeys() {
		if isPossibleMatch(badID, key.String()) {
			matches = append(matches, key.String())
		}
	}

	return
}

func isPossibleMatch(badID, testID string) bool {
	if badID == testID {
		return false
	}

	badBase := path.Base(badID)
	testBase := path.Base(testID)

	if badBase == testBase {
		return true
	} else if strings.Contains(testBase, badBase) || strings.Contains(badBase, testBase) &&
		math.Abs(float64(len(testBase))-float64(len(badBase))) < 6 {
		return true
	}

	return false
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
