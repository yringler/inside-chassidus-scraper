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
		colly.AllowedDomains("insidechassidus.org"),
	)

	scraper.collector.OnError(func(r *colly.Response, err error) {
		fmt.Println("Scrape error: " + err.Error())
		fmt.Println("Url: ", r.Request.URL.RawPath)
	})

	// Scrape the top level sections.
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
		parent := e.DOM.Parent()
		title := parent.Find("h1").Text()
		description := parent.Find("div").Text()
		mp3, _ := parent.Find("a[mp3]").Attr("mp3")

		scraper.activeSection.Lessons = append(scraper.activeSection.Lessons, Lesson{
			Title:       title,
			Description: description,
			Audio:       []string{mp3},
		})
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

	scraper.activeSection.Lessons = append(scraper.activeSection.Lessons, Lesson{
		Title:       name,
		Description: description,
		Audio:       audio,
		Pdf:         pdfs,
	})
}

func (scraper *InsideScraper) loadSection(firstColumn, domDescription *goquery.Selection) {
	// This method creates a new section, and changes the active section to it.
	// After it is finished, it restores the active session to the parent.
	parentOfNewSection := scraper.activeSection

	// The name of the section. A link.
	domName := firstColumn.Find("a")

	sectionURL, err := getSectionURL(firstColumn, domDescription)

	if err != nil {
		panic(err)
	}

	newSection := &SiteSection{
		Title:       domName.Text(),
		Description: domDescription.Text(),
		Sections:    make([]SiteSection, 0, 20),
		Lessons:     make([]Lesson, 0, 20),
	}

	// In general, the active section will always be set by the primary menu, so this will
	// never be nil.
	// But if we're debugging one page, it would be. Hence the test.
	if parentOfNewSection != nil {
		parentOfNewSection.Sections = append(parentOfNewSection.Sections, *newSection)

		newSection = &parentOfNewSection.Sections[len(parentOfNewSection.Sections)-1]
	}

	scraper.activeSection = newSection

	scraper.collector.Visit(sectionURL)

	// For debugging, to support testing a particular section.
	// remember to comment out when your finished.
	// TODO: add a CLI argument to specify url, handle this automatically.
	//scraper.site = append(scraper.site, *newSection)

	// Restore active section.
	scraper.activeSection = parentOfNewSection
}

// Gets the URL where content in this section is located.
func getSectionURL(firstColumn, domDescription *goquery.Selection) (string, error) {
	url, err := getSectionURLFromHereLink(firstColumn, domDescription)

	if url != "" {
		return url, nil
	} else if err != nil {
		fmt.Println(err)
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
