package insidescraper

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
	// To keep track of redirects. (We can compare the requested URL with where we actually
	// ended up).
	requestingURL string
	// Keep track of URLs which are redirecting. The key is the original URL, value is final URL
	redirects map[string]string
	site      map[string]SiteSection
	collector *colly.Collector
}

// Site returns the data which represents the site/lesson structure.
func (scraper *InsideScraper) Site() []SiteSection {
	site := make([]SiteSection, 0, len(scraper.site))

	for _, value := range scraper.site {
		site = append(site, value)
	}

	return site
}

// Scrape scrapes the site. It returns an error if there's an error.
func (scraper *InsideScraper) Scrape() (err error) {
	scraper.site = make(map[string]SiteSection, 1000)
	scraper.redirects = make(map[string]string, 100)

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
		fmt.Println("(Possibly) related Url: ", scraper.activeSection)
	})

	scraper.collector.OnRequest(func(request *colly.Request) {
		// If a redirect happened, register it.
		//if scraper.requestingURL != "" && scraper.requestingURL != request.URL.String() {
		//		scraper.redirects[scraper.requestingURL] = request.URL.String()
		//		}
	})

	// Scrape the top level sections.
	scraper.collector.OnHTML("body.home #main-menu-fst > li > a ", func(e *colly.HTMLElement) {
		sectionURL := sanatizeURL(e.Attr("href"))
		sectionID := getHash(sectionURL)

		// If a top level section was already scraped as a sub section, simply
		// mark it as being a top level section.
		if section, exists := scraper.site[sectionID]; exists {
			section.IsTopLevel = true
			scraper.site[sectionID] = section
			return
		}

		scraper.site[sectionID] = SiteSection{
			ID:         sectionID,
			Title:      e.Text,
			IsTopLevel: true,
			Sections:   make([]string, 0, 100),
		}

		scraper.activeSection = sectionID
		err := scraper.collector.Visit(sectionURL)

		if err != nil {
			fmt.Println("Visit main section error (", sectionID, "):", err.Error())
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
				descriptionColumn = secondColumn;
			}
			scraper.loadSection(firstColumn, descriptionColumn)
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
	scraper.collector.Visit("https://insidechassidus.org/thought-and-history")

	scraper.resolveRedirects()

	return err
}

// If a redirect happens, all references to the bad URL need to be updated to the correct URL.
func (scraper *InsideScraper) resolveRedirects() {

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

	sectionURLs, err := scraper.getSectionURLs(firstColumn, domDescription)

	if err != nil {
		panic(err)
	}

	// Generate sub section from description with multiple urls.
	if len(sectionURLs) > 1 {
		subSections := make([]string, 0, len(sectionURLs))
		for _, url := range sectionURLs {
			subSections = append(subSections, getHash(url))
		}

		currentURL, _ :=  domName.Attr("href")
		currentURL = sanatizeURL(currentURL)
		currentID := getHash(currentURL)

		if _, exists := scraper.site[currentID]; exists {
			panic("Error!!! Section which references other sections already exists!!!\nParent:" +
				scraper.activeSection + "\nAlready here error cause: " + currentID)
		}

		scraper.site[currentID] = SiteSection{
			ID:          currentID,
			Title:       domName.Text(),
			Description: domDescription.Text(),
			Sections:    subSections,
		}

		activeSection := scraper.site[scraper.activeSection]
		activeSection.Sections = append(activeSection.Sections, currentID)
		scraper.site[scraper.activeSection] = activeSection

		return
	}

	sectionURL := sectionURLs[0]

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
	scraper.requestingURL = sectionURL
	scraper.collector.Visit(sectionURL)

	scraper.activeSection = parentOfNewSection
}

// Gets the URL where content in this section is located.
func (scraper *InsideScraper) getSectionURLs(firstColumn, domDescription *goquery.Selection) ([]string, error) {
	urls := getSectionURLFromHereLink(firstColumn, domDescription)

	if urls != nil {
		return urls, nil
	}

	url, err := getSectionURLFromTitle(firstColumn)

	if url != "" {
		return []string{url}, nil
	} else if err != nil {
		return nil, err
	}

	rowHTML, _ := firstColumn.Closest("tr").Html()
	return nil, errors.New("error: section url not found. DOM: \n" + rowHTML)
}

// Some sections have the correct URL to its contents in a here link in the description.
func getSectionURLFromHereLink(firstColumn, domDescription *goquery.Selection) []string {
	hereLink := domDescription.Find("a").FilterFunction(func(i int, selection *goquery.Selection) bool {
		url, _ := selection.Attr("href")

		return strings.Contains(selection.Text(), "here") && strings.Contains(url, "insidechassidus.org")
	})

	if hereLink.Length() == 0 {
		return nil
	}

	return hereLink.Map(func(_ int, selection *goquery.Selection) string {
		if url, exists := selection.Attr("href"); exists {
			return sanatizeURL(url)
		}

		panic("Hey! No url")
	})
}

// Most sections have the URL to contents in the title (which is a link).
func getSectionURLFromTitle(firstColumn *goquery.Selection) (string, error) {

	sectionURL, exists := firstColumn.Find("a").Attr("href")
	if !exists {
		parentHTML, _ := firstColumn.Closest("tr").Html()
		childHTML, _ := firstColumn.Html()
		return "", errors.New("No href!\nParent: " + parentHTML + "\nchild:\n" + childHTML)
	}

	return sanatizeURL(sectionURL), nil
}

func getHash(source string) string {
	// There should never be www. in URL because it redirects.
	source = sanatizeURL(source)
	//	idBytes := md5.Sum([]byte(source))
	//	return fmt.Sprintf("%x", idBytes)
	return source
}

func sanatizeURL(href string) string {
	return strings.Replace(href, "www.", "", 1)
}

func isOnMobile(dom *goquery.Selection) bool {
	return dom.Closest(".visible-xs").Length() != 0
}
