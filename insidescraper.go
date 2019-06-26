package main

import (
	"errors"
	"fmt"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
)

// InsideScraper scrapes insidechassidus for lesson structure.
type InsideScraper struct {
	activeSection *SiteSection
	site          []SiteSection
	collector     *colly.Collector
}

// Site returns the data which represents the site/lesson structure.
func (scraper *InsideScraper) Site() []SiteSection {
	return scraper.site
}

// Scrape scrapes the site. It returns an error if there's an error.
func (scraper *InsideScraper) Scrape() (err error) {
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
		colly.AllowedDomains("http://insidechassidus.org"),
	)

	scraper.collector.OnError(func(r *colly.Response, err error) {
		fmt.Println("Scrape error: " + err.Error())
		fmt.Println("Url: ", r.Request.URL.RawPath)
	})

	scraper.collector.OnHTML("body.home #main-menu-fst > li > a ", func(e *colly.HTMLElement) {
		scraper.site = append(scraper.site, SiteSection{
			Title:    e.Text,
			Sections: make([]SiteSection, 0, 20),
		})

		scraper.activeSection = &scraper.site[len(scraper.site)-1]
		err := scraper.collector.Visit(e.Attr("href"))

		if err != nil {
			fmt.Println("Visit main section error: " + err.Error())
		}
	})

	scraper.collector.OnHTML("tbody tr", func(e *colly.HTMLElement) {
		domParent := e.DOM

		firstColumn := domParent.Find("td:nth-child(1)")
		secondColumn := domParent.Find("td:nth-child(2)")
		thirdColumn := domParent.Find("td:nth-child(3)")

		// If this row has only 2 columns and it doesn't describe a class
		// than it must be a section.
		if thirdColumn.Length() == 0 && secondColumn.Find("[mp3],a[href$=\".pdf\"]").Length() == 0 {
			scraper.loadSection(firstColumn, secondColumn)
		}
	})

	scraper.collector.Visit("http://insidechassidus.org/")

	return err
}

func (scraper *InsideScraper) loadSection(firstColumn, domDescription *goquery.Selection) {
	// This method creates a new section, and changes the active section to it.
	// After it is finished, it restores the active session to the parent.
	parentOfNewSection := scraper.activeSection

	// The name of the section. A link.
	domName := firstColumn.Find("a")

	sectionURL, exists := domName.Attr("href")
	if !exists {
		parentHTML, _ := firstColumn.Closest("tr").Html()
		childHTML, _ := firstColumn.Html()
		panic("Error! no href!\nParent: " + parentHTML + "\nchild:\n" + childHTML)
	}

	parentOfNewSection.Sections = append(parentOfNewSection.Sections, SiteSection{
		Title:       domName.Text(),
		Description: domDescription.Text(),
		Sections:    make([]SiteSection, 0, 20),
		Lessons:     make([]Lesson, 0, 20),
	})

	newSection := &parentOfNewSection.Sections[len(parentOfNewSection.Sections)-1]

	scraper.activeSection = newSection

	scraper.collector.Visit(sectionURL)

	// Restore active section.
	scraper.activeSection = parentOfNewSection
}
