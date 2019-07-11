package insidescraper

import (
	"io/ioutil"
	"math"
	"net/http"
	"path"
	"strings"
)

// PostScraper goes over the scraped data and fixes it up as much as possible.
type PostScraper struct {
	Site    map[string]SiteSection
	Missing map[string]Correction
	Empty   map[string]Correction
}

// Correction is a (possible) correction for a missing link.
type Correction struct {
	Guesses      []string
	Parent       string
	Is404        bool
	IsConfirmed  bool
	WasCorrected bool
}

// FixSite applies fixes to the site data.
func (cleaner *PostScraper) FixSite() {
	cleaner.Missing = cleaner.GetMissingCorrections()
	cleaner.Empty = cleaner.GetMissingCorrections()

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

	for parentID, section := range cleaner.Site {
		for _, subSectionID := range section.Sections {
			// Don't find correction twice.
			if _, exists := corrections[subSectionID]; exists {
				//continue
			}

			if _, exists := cleaner.Site[subSectionID]; !exists {
				corrections[subSectionID] = cleaner.getPossibleMatches(subSectionID, parentID)
			}
		}
	}

	return corrections
}

// GetEmptyCorrections finds all empty sections (no lessons or subsections) and tries to correct.
func (cleaner *PostScraper) GetEmptyCorrections() map[string]Correction {
	corrections := make(map[string]Correction, 10)

	for parentID, section := range cleaner.Site {
		for _, subSectionID := range section.Sections {
			// Don't try to correct the same thing twice.
			if _, exists := corrections[subSectionID]; exists {
				continue
			}

			if subSection, exists := cleaner.Site[subSectionID]; exists {
				if len(subSection.Sections) == 0 && len(subSection.Lessons) == 0 {
					corrections[subSectionID] = cleaner.getPossibleMatches(subSectionID, parentID)
				}
			}
		}
	}

	return corrections
}

// applyFix fixes up the site based on the correction. If the correction is executed, marked as such.
func (cleaner *PostScraper) applyFix(badID string, correction *Correction) {
	if len(correction.Guesses) > 0 && (correction.IsConfirmed || correction.Is404) {
		if _, exists := cleaner.Site[badID]; exists {
			delete(cleaner.Site, badID)
		}

		for _, section := range cleaner.Site {
			for i, subSectionID := range section.Sections {
				if subSectionID == badID {
					section.Sections[i] = correction.Guesses[0]
				}
			}
		}

		correction.WasCorrected = true
	}
}

func (cleaner *PostScraper) getPossibleMatches(id, parentID string) Correction {
	correction := Correction{
		Parent: parentID,
	}

	if response, err := http.Head(id); err == nil {
		if response.StatusCode == http.StatusNotFound {
			correction.Is404 = true
		}
	}

	correction.Guesses = cleaner.getPossibleIds(id)

	if correction.Guesses != nil {
		body1 := getBody(id)
		body2 := getBody(correction.Guesses[0])

		if body1 != "" && body1 == body2 {
			correction.IsConfirmed = true
		}
	}

	return correction
}

func (cleaner *PostScraper) getPossibleIds(id string) []string {
	matches := make([]string, 0, 10)
	idBase := path.Base(id)

	for key := range cleaner.Site {
		if key == id {
			continue
		}

		keyBase := path.Base(key)
		if keyBase == idBase {
			matches = append(matches, key)
		} else if strings.Contains(keyBase, idBase) ||
			strings.Contains(idBase, keyBase) &&
				math.Abs(float64(len(keyBase))-float64(len(idBase))) < 6 {
			matches = append(matches, key)
		}
	}

	if len(matches) > 0 {
		return matches
	}

	return nil
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
