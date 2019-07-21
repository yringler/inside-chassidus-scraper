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

	c.Visit("https://insidechassidus.org/maamarim/maamarim-of-the-rebbe/text-based-concise-summary/1553-maamarim-5715")
}
