package main

import (
	"errors"
	"github.com/gocolly/colly"
)

// InsideScraper scrapes insidechassidus for lesson structure.
type InsideScraper struct {
	activeSection *SiteSection
	site []SiteSection
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

	c := colly.NewCollector(
		colly.UserAgent("inside-scraper"),
	)

	c.OnError(func(_ *colly.Response, err error) {
		panic("Scrape error: " + err.Error())
	})

	c.OnHTML("body.home #main-menu-fst > li > a ", func(e *colly.HTMLElement) {
		scraper.site = append(scraper.site, SiteSection{
			Title:    e.Text,
			Sections: make([]SiteSection, 0, 20),
		})

		scraper.activeSection = &scraper.site[len(scraper.site)-1]
		err := c.Visit(e.Attr("href"))

		if err != nil {
			panic("Vist main section error: " + err.Error())
		}
	})

	// Process sub sections
	c.OnHTML("tbody tr td:nth-child(2)", func(e *colly.HTMLElement) {
		parentOfCurrentSection := scraper.activeSection

		domParent := e.DOM.Closest("tr")
		domName := domParent.Find("td:first-child a")
		domDescription := domParent.Find("td:nth-child(2)")

		parentOfCurrentSection.Sections = append(parentOfCurrentSection.Sections, SiteSection{
			Title:       domName.Text(),
			Description: domDescription.Text(),
			Sections:    make([]SiteSection, 0, 20),
			Lessons:     make([]Lesson, 0, 20),
		})

		scraper.activeSection = &parentOfCurrentSection.Sections[len(parentOfCurrentSection.Sections)-1]
		sectionURL, exists := domName.Attr("href")

		if !exists {
			parentHTML, _ := domParent.Html()
			childHTML, _ := domName.Html()
			panic("Error! no href!\nParent: " + parentHTML + "\nchild:\n" + childHTML)
		}

		c.Visit(sectionURL)
		scraper.activeSection = parentOfCurrentSection
	})

	c.Visit("http://insidechassidus.org/")

	return err
}