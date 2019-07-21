package insidescraper

import (
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
)

// InsideScraper scrapes insidechassidus for lesson structure.
type InsideScraper struct {
	activeSection string
	Site          Site
	collector     *colly.Collector
	// If a section ends up being a lesson (ie it only has lessons, of < 2 audio each)
	// keep track of it.
	// Maps the original secion id to the lesson id, so that further references to the section
	// can get the lesson.
	sectionLessons map[string]string
}

// Scrape scrapes the site. It returns an error if there's an error.
func (scraper *InsideScraper) Scrape(scrapeURL ...string) (err error) {
	scraper.Site.Sections = make(map[string]SiteSection, 1000)
	scraper.Site.Lessons = make(map[string]Lesson, 1000)
	scraper.Site.TopLevel = make([]string, 0, 10)
	scraper.sectionLessons = make(map[string]string)

	// defer func() {
	// 	if r := recover(); r != nil {
	// 		switch x := r.(type) {
	// 		case string:
	// 			err = errors.New(x)
	// 		case error:
	// 			err = x
	// 		default:
	// 			err = errors.New("Unknown panic in Scrape()")
	// 		}
	// 	}
	// }()

	scraper.collector = colly.NewCollector(
		colly.UserAgent("inside-scraper"),
		colly.AllowedDomains("insidechassidus.org"),
	)

	scraper.collector.OnError(func(r *colly.Response, err error) {
		fmt.Fprintln(os.Stderr, "Scrape error: "+err.Error())
		fmt.Fprintln(os.Stderr, "(Possibly) related Url: ", scraper.activeSection+"\n")
	})

	// Scrape the top level sections.
	scraper.collector.OnHTML("body.home #main-menu-fst > li > a ", func(e *colly.HTMLElement) {
		sectionURL := getFinalURL(e.Attr("href"))
		sectionID := getHash(sectionURL)

		// If a top level section was already scraped as a sub section, simply
		// mark it as being a top level section.
		if _, exists := scraper.Site.Sections[sectionID]; exists {
			scraper.Site.TopLevel = append(scraper.Site.TopLevel, sectionID)
			return
		}

		scraper.Site.Sections[sectionID] = SiteSection{
			SiteData: &SiteData{
				Title: e.Text,
			},
			ID:       sectionID,
			Sections: make([]string, 0, 100),
		}
		scraper.Site.TopLevel = append(scraper.Site.TopLevel, sectionID)

		scraper.activeSection = sectionID
		err := scraper.collector.Visit(sectionURL)

		if err != nil {
			fmt.Fprintln(os.Stderr, "Visit main section error (", sectionID, "):", err.Error()+"\n")
		}
	})

	// Scrape lessons and sub sections.
	scraper.collector.OnHTML("tbody tr", func(e *colly.HTMLElement) {
		domParent := e.DOM

		firstColumn := domParent.Find("td:nth-child(1)")
		secondColumn := domParent.Find("td:nth-child(2)")
		thirdColumn := domParent.Find("td:nth-child(3)")

		// If there's no media in the second column, then it must be a section.
		if secondColumn.Find("[mp3],a[href$=\".pdf\"]").Length() == 0 {
			// Note that sometimes (Eg Rebbetzin Shaindle https://insidechassidus.org/thought-and-history/23-lives-of-the-chabad-rebbeim)
			// a section is shown as a lesson without media, so the columns are title | (blank) | description.
			// If that's the case, use the 3rd column as the description.

			descriptionColumn := thirdColumn
			if descriptionColumn.Length() == 0 {
				descriptionColumn = secondColumn
			}
			scraper.loadSection(firstColumn, descriptionColumn)
		} else if thirdColumn.Length() != 0 {
			scraper.loadLessons(domParent)
		} else {
			text, _ := domParent.Html()
			fmt.Fprintln(os.Stderr, "Error: could not process row", text)
		}
	})

	// Scrape lessons which aren't in a table
	scraper.collector.OnHTML("div > div > a[mp3]", func(e *colly.HTMLElement) {
		if isOnMobile(e.DOM) {
			return
		}

		parent := e.DOM.Parent()
		title := parent.Find("h1").Text()
		description := parent.Find("div").Text()
		mp3, _ := parent.Find("a[mp3]").Attr("mp3")

		newLesson := Lesson{
			ID: strconv.Itoa(rand.Int()),
			SiteData: &SiteData{
				Title:       title,
				Description: description,
			},
			Audio: []Media{Media{
				Source: mp3,
			},
			},
		}

		scraper.Site.Lessons[newLesson.ID] = newLesson

		activeSection, _ := scraper.Site.Sections[scraper.activeSection]
		activeSection.Lessons = append(activeSection.Lessons, newLesson.ID)
		scraper.Site.Sections[scraper.activeSection] = activeSection
	})

	source := "https://insidechassidus.org/"
	if len(scrapeURL) == 1 {
		source = scrapeURL[0]
	}

	scraper.collector.Visit(source)

	scraper.applyLessonConversions()

	return err
}

func (scraper *InsideScraper) loadLessons(dom *goquery.Selection) {
	lessonScraper := LessonScraper{
		Row: dom,
	}

	lessonScraper.LoadLesson()
	scraper.Site.Lessons[lessonScraper.Lesson.ID] = *lessonScraper.Lesson

	// Append this lesson id to current section.
	activeSection, _ := scraper.Site.Sections[scraper.activeSection]
	activeSection.Lessons = append(activeSection.Lessons, lessonScraper.Lesson.ID)
	scraper.Site.Sections[scraper.activeSection] = activeSection
}

func (scraper *InsideScraper) loadSection(firstColumn, domDescription *goquery.Selection) {

	// The name of the section. A link.
	domName := firstColumn.Find("a")

	sectionURLs := getSectionURLFromHereLink(domDescription)

	sectionTitleURL, err := scraper.getSectionURLFromTitle(firstColumn)

	// Urls in description are references to sections which are in a different section.
	// Don't scrape them now, just add that reference to the current section.

	// More than 1: create a sub section.
	if len(sectionURLs) > 1 {
		subSections := make([]string, 0, len(sectionURLs))
		for _, url := range sectionURLs {
			subSections = append(subSections, getHash(url))
		}

		currentURL, _ := domName.Attr("href")
		currentURL = getFinalURL(currentURL)
		currentID := currentURL

		if _, exists := scraper.Site.Sections[currentID]; exists {
			panic("Error!!! Section which references other sections already exists!!!\nParent:" +
				scraper.activeSection + "\nAlready here error cause: " + currentID)
		}

		scraper.Site.Sections[currentID] = SiteSection{
			SiteData: &SiteData{
				Title:       domName.Text(),
				Description: domDescription.Text(),
			},
			ID:       currentID,
			Sections: subSections,
		}

		activeSection := scraper.Site.Sections[scraper.activeSection]
		activeSection.Sections = append(activeSection.Sections, currentID)
		scraper.Site.Sections[scraper.activeSection] = activeSection

		return
	}

	// If there's only 1 referenced: Add it to current section.
	// If description has same URL as title, then this is a good link and we should follow it.
	if len(sectionURLs) == 1 && sectionURLs[0] != sectionTitleURL {
		activeSection := scraper.Site.Sections[scraper.activeSection]
		activeSection.Sections = append(activeSection.Sections, getHash(sectionURLs[0]))
		scraper.Site.Sections[scraper.activeSection] = activeSection

		return
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error()+"\n")
		return
	}

	if sectionTitleURL == "" {
		fmt.Fprintln(os.Stderr, "Error: URL not found. Parent: "+scraper.activeSection)
		return
	}

	sectionID := getHash(sectionTitleURL)

	// Add this section to the parent section.
	if scraper.activeSection != "" {
		activeSection, _ := scraper.Site.Sections[scraper.activeSection]
		activeSection.Sections = append(activeSection.Sections, sectionID)
		scraper.Site.Sections[scraper.activeSection] = activeSection
	}

	// If a section is referenced in multiple places and it was already visited,
	// don't try to create it again; it'll end up making an empty section (because it's URL has
	// already been scraped), and over-writing the real data.
	if _, hasKey := scraper.Site.Sections[sectionID]; hasKey {
		return
	}

	/*
		Load a section and all of it's children.
	*/

	newSection := SiteSection{
		SiteData: &SiteData{
			Title:       domName.Text(),
			Description: domDescription.Text(),
		},
		ID:       sectionID,
		Sections: make([]string, 0, 20),
		Lessons:  make([]string, 0, 20),
	}

	scraper.Site.Sections[sectionID] = newSection

	// Back up current section, restore it as active after finished with this section.
	parentOfNewSection := scraper.activeSection
	scraper.activeSection = sectionID
	scraper.collector.Visit(sectionTitleURL)

	// If this section is really a lesson, save that fact for later use.
	if err := scraper.Site.ConvertToLesson(sectionID); err != nil {
		scraper.sectionLessons[sectionID] = sectionID
	}

	scraper.activeSection = parentOfNewSection
}

// Some sections have the correct URL to its contents in a here link in the description.
func getSectionURLFromHereLink(domDescription *goquery.Selection) []string {
	hereLink := domDescription.Find("a").FilterFunction(func(i int, selection *goquery.Selection) bool {
		url, _ := selection.Attr("href")

		return strings.Contains(selection.Text(), "here") && strings.Contains(url, "insidechassidus.org")
	})

	if hereLink.Length() == 0 {
		return nil
	}

	return hereLink.Map(func(_ int, selection *goquery.Selection) string {
		if url, exists := selection.Attr("href"); exists {
			return getFinalURL(url)
		}

		panic("Hey! No url")
	})
}

// Most sections have the URL to contents in the title (which is a link).
func (scraper *InsideScraper) getSectionURLFromTitle(firstColumn *goquery.Selection) (string, error) {

	sectionURL, exists := firstColumn.Find("a").Attr("href")
	if !exists {
		childHTML, _ := firstColumn.Html()
		return "", errors.New("No href!\nParent: " + scraper.activeSection + "\nchild:\n" + childHTML)
	}

	return getFinalURL(sectionURL), nil
}

// If a section was converted to a lesson, there may be references to that section.
// Update them to refer to the lesson.
func (scraper *InsideScraper) applyLessonConversions() {
	// Go through every section.
	for _, section := range scraper.Site.Sections {
		tmpSections := section.Sections[:0]

		// Go through every child.
		for _, childSectionID := range section.Sections {
			// Handle sections being converted to lessons.
			lessonID, exists := scraper.sectionLessons[childSectionID]
			if exists {
				section.Lessons = append(section.Lessons, lessonID)
			} else {
				tmpSections = append(tmpSections, childSectionID)
			}
		}
	}
}

// Get's the URL after all redirects.
func getFinalURL(url string) string {
	response, err := http.Head(url)
	if err == nil {
		return response.Request.URL.String()
	}

	fmt.Fprintln(os.Stderr, "Error: failed HEAD request ("+url+")\nError: "+err.Error()+"\n")
	return url
}

func getHash(source string) string {
	//	idBytes := md5.Sum([]byte(source))
	//	return fmt.Sprintf("%x", idBytes)
	return source
}

func isOnMobile(dom *goquery.Selection) bool {
	return dom.Closest(".visible-xs").Length() != 0
}
