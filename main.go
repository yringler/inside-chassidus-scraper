package main

import (
	"encoding/json"
	"fmt"

	"github.com/gocolly/colly"
)

func main() {
	site := make([]SiteSection, 5, 20)
	var activeSection *SiteSection

	c := colly.NewCollector(
		colly.UserAgent("inside-scraper"),
	)

	c.OnError(func(_ *colly.Response, err error) {
		fmt.Println("Something went wrong:", err)
	})

	c.OnHTML("body.home #main-menu-fst > li > a ", func(e *colly.HTMLElement) {
		site = append(site, SiteSection{
			Title:    e.Text,
			Sections: make([]SiteSection, 0, 20),
		})

		activeSection = &site[len(site)-1]
		err := c.Visit(e.Attr("href"))

		if err != nil {
			panic(err)
		}
	})

	// Process sub sections
	c.OnHTML("tbody tr td:nth-child(2)", func(e *colly.HTMLElement) {
		parentOfCurrentSection := activeSection

		domParent := e.DOM.Closest("tr")
		domName := domParent.Find("td:first-child a")
		domDescription := domParent.Find("td:nth-child(2)")

		parentOfCurrentSection.Sections = append(parentOfCurrentSection.Sections, SiteSection{
			Title:       domName.Text(),
			Description: domDescription.Text(),
			Sections:    make([]SiteSection, 0, 20),
			Lessons:     make([]Lesson, 0, 20),
		})

		activeSection = &parentOfCurrentSection.Sections[len(parentOfCurrentSection.Sections)-1]
		sectionURL, exists := domName.Attr("href")

		if !exists {
			parentHtml, _ := domParent.Html()
			childHtml, _ := domName.Html()
			panic("Error! no href!\nParent: " + parentHtml + "\nchild:\n" + childHtml)
		}

		c.Visit(sectionURL)
		activeSection = parentOfCurrentSection
	})

	// Process lessons (which have a 3rd column for media)
	/*
		c.OnHTML("tbody tr td:nth-child(3)", func(e *colly.HTMLElement) {
			parent := e.DOM.Closest("tr")
		})
	*/

	c.OnScraped(func(_ *colly.Response) {
		text, _ := json.Marshal(site)
		fmt.Println(string(text))
	})

	c.Visit("http://insidechassidus.org/")
}
