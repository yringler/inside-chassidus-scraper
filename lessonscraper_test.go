package insidescraper

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/gocolly/colly"
)

func TestExtractCompositeLessons(t *testing.T) {
	c := colly.NewCollector(
		colly.UserAgent("inside-scraper"),
		colly.AllowedDomains("insidechassidus.org"),
	)

	c.OnHTML("tbody tr", func(e *colly.HTMLElement) {
		scraper := LessonScraper{
			Row: e.DOM,
		}
		scraper.LoadLesson()

		jsonOut, _ := json.MarshalIndent(*scraper.Lesson, "", "    ")
		fmt.Println(string(jsonOut))
	})
	fmt.Println("Test1")
	c.Visit("https://insidechassidus.org/maamarim/maamarim-of-the-rebbe/text-based-concise-summary/1553-maamarim-5715")
	fmt.Println("Test2")
	c.Visit("https://insidechassidus.org/maamarim/maamarim-of-the-rebbe/text-based-concise-summary/1553-maamarim-5715")
}

// Check that sections which are only lessons are handled right.
func TestScrapeSimpleSection(t *testing.T) {
	runScraper("https://insidechassidus.org/jewish-and-chassidic-calendar/shvat/217-15-shvat")
}

// Check output of parsing master page containing many simple sections.
func TestScrapeParentSimpleSection(t *testing.T) {
	runScraper("https://insidechassidus.org/thought-and-history/123-kabbala-and-philosophy-series")
}

// Intense PDF course.
func TestPdfCrash(t *testing.T) {
	runScraper("https://insidechassidus.org/maamarim/maamarim-of-the-rebbe/text-based-concise-summary")
}
