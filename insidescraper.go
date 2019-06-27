package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
)

// InsideScraper scrapes insidechassidus for lesson structure.
type InsideScraper struct {
	activeSection string
	site          map[string]SiteSection
	collector     *colly.Collector
}

// Site returns the data which represents the site/lesson structure.
func (scraper *InsideScraper) Site() map[string]SiteSection {
	return scraper.site
}

// Scrape scrapes the site. It returns an error if there's an error.
func (scraper *InsideScraper) Scrape() (err error) {
	scraper.site = make(map[string]SiteSection, 1000)

	defer func() {
		if r := recover(); r != nil {
			switch x := r.(type) {
			case string:
				err = errors.New(x)
			case error:
				err = x
			default:
				err = errors.New("Unknown panic in Scrape()")
			}
		}
	}()

	scraper.collector = colly.NewCollector(
		colly.UserAgent("inside-scraper"),
		colly.AllowedDomains("insidechassidus.org"),
	)

	scraper.collector.OnError(func(r *colly.Response, err error) {
		fmt.Println("Scrape error: " + err.Error())
		fmt.Println("Url: ", r.Request.URL.RawPath)
	})

	// Scrape the top level sections.
	scraper.collector.OnHTML("body.home #main-menu-fst > li > a ", func(e *colly.HTMLElement) {
		sectionURL := e.Attr("href")
		sectionID := getHash(sectionURL)

		scraper.site[sectionID] = SiteSection{
			ID:         getHash(sectionURL),
			Title:      e.Text,
			IsTopLevel: true,
			Sections:   make([]string, 0, 100),
		}

		scraper.activeSection = sectionID
		err := scraper.collector.Visit(sectionURL)

		if err != nil {
			fmt.Println("Visit main section error: " + err.Error())
		}
	})

	// Scrape lessons and sub sections.
	scraper.collector.OnHTML("tbody tr", func(e *colly.HTMLElement) {
		domParent := e.DOM

		firstColumn := domParent.Find("td:nth-child(1)")
		secondColumn := domParent.Find("td:nth-child(2)")
		thirdColumn := domParent.Find("td:nth-child(3)")

		// If this row has only 2 columns and it doesn't describe a class
		// than it must be a section.
		if thirdColumn.Length() == 0 && secondColumn.Find("[mp3],a[href$=\".pdf\"]").Length() == 0 {
			scraper.loadSection(firstColumn, secondColumn)
		} else if thirdColumn.Length() != 0 {
			scraper.loadLessons(firstColumn, secondColumn, thirdColumn)
		} else {
			text, _ := domParent.Html()
			fmt.Println("Error: could not process row", text)
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

		activeSection, _ := scraper.site[scraper.activeSection]
		activeSection.Lessons = append(activeSection.Lessons, Lesson{
			Title:       title,
			Description: description,
			Audio:       []string{mp3},
		})

		scraper.site[scraper.activeSection] = activeSection
	})

	// Take it from the top.
	scraper.collector.Visit("http://insidechassidus.org/")

	return err
}

func (scraper *InsideScraper) loadLessons(domName, domMedia, domDescription *goquery.Selection) {
	name := domName.Text()
	description := domDescription.Text()

	pdfs := make([]Media, 0)

	domMedia.Find("a[href$=\".pdf\"]").Each(func(_ int, selection *goquery.Selection) {
		source, _ := selection.Attr("href")

		pdfs = append(pdfs, Media{
			Title:  selection.Text(),
			Source: source,
		})
	})

	audio := domMedia.Find("[mp3]").Map(func(_ int, selection *goquery.Selection) string {
		value, _ := selection.Attr("mp3")
		return value
	})

	activeSection, _ := scraper.site[scraper.activeSection]
	activeSection.Lessons = append(activeSection.Lessons, Lesson{
		Title:       name,
		Description: description,
		Audio:       audio,
		Pdf:         pdfs,
	})
	scraper.site[scraper.activeSection] = activeSection
}

func (scraper *InsideScraper) loadSection(firstColumn, domDescription *goquery.Selection) {

	// The name of the section. A link.
	domName := firstColumn.Find("a")

	sectionURL, err := scraper.getSectionURL(firstColumn, domDescription)

	if err != nil {
		panic(err)
	}

	sectionID := getHash(sectionURL)

	// Add this section to the parent section.
	if scraper.activeSection != "" {
		activeSection, _ := scraper.site[scraper.activeSection]
		activeSection.Sections = append(activeSection.Sections, sectionID)
		scraper.site[scraper.activeSection] = activeSection
	}

	// If a section is referenced in multiple places and it was already visited,
	// don't try to create it again; it'll end up making an empty section (because it's URL has
	// already been scraped), and over-writing the real data.
	if _, hasKey := scraper.site[sectionID]; hasKey {
		return
	}

	/*
		Load a section and all of it's children.
	*/

	newSection := SiteSection{
		Title:       domName.Text(),
		ID:          sectionID,
		Description: domDescription.Text(),
		Sections:    make([]string, 0, 20),
		Lessons:     make([]Lesson, 0, 20),
	}

	scraper.site[sectionID] = newSection

	// Back up current section, restore it as active after finished with this section.
	parentOfNewSection := scraper.activeSection
	scraper.activeSection = sectionID
	scraper.collector.Visit(sectionURL)
	scraper.activeSection = parentOfNewSection
}

// Gets the URL where content in this section is located.
func (scraper *InsideScraper) getSectionURL(firstColumn, domDescription *goquery.Selection) (string, error) {
	url, err := getSectionURLFromHereLink(firstColumn, domDescription)

	if url != "" {
		return url, nil
	} else if err != nil {
		fmt.Println("Error in getSectionURL (" + scraper.activeSection + ")\n" + err.Error())
	}

	url, err = getSectionURLFromTitle(firstColumn, domDescription)

	if url != "" {
		return url, nil
	} else if err != nil {
		return "", err
	}

	rowHTML, _ := firstColumn.Closest("tr").Html()
	return "", errors.New("error: section url not found. DOM: \n" + rowHTML)
}

// Some sections have the correct URL to its contents in a here link in the description.
func getSectionURLFromHereLink(firstColumn, domDescription *goquery.Selection) (string, error) {
	hereLink := domDescription.Find("a").FilterFunction(func(i int, selection *goquery.Selection) bool {
		return strings.Contains(selection.Text(), "here")
	})

	if hereLink.Length() > 1 {
		descriptionHTML, _ := domDescription.Html()
		return "", errors.New("Too many here links\n" + descriptionHTML + "\n\n")
	} else if hereLink.Length() == 1 {
		sectionURL, exists := hereLink.Attr("href")

		if !exists {
			parentHTML, _ := domDescription.Closest("tr").Html()
			childHTML, _ := domDescription.Html()
			return "", errors.New("No href!\nParent: " + parentHTML + "\nchild:\n" + childHTML)
		}

		return sectionURL, nil
	}

	// Nothing found, but not an error. The description doesn't have to contain a link.
	return "", nil
}

// Most sections have the URL to contents in the title (which is a link).
func getSectionURLFromTitle(firstColumn, domDescription *goquery.Selection) (string, error) {

	sectionURL, exists := firstColumn.Find("a").Attr("href")
	if !exists {
		parentHTML, _ := firstColumn.Closest("tr").Html()
		childHTML, _ := firstColumn.Html()
		return "", errors.New("No href!\nParent: " + parentHTML + "\nchild:\n" + childHTML)
	}

	return sectionURL, nil
}

func getHash(source string) string {
	//	idBytes := md5.Sum([]byte(source))
	//	return fmt.Sprintf("%x", idBytes)
	return source
}

func isOnMobile(dom *goquery.Selection) bool {
	return dom.Closest(".visible-xs").Length() != 0
}
