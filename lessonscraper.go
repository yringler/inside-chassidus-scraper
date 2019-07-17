package insidescraper

import (
	"strings"
	"unicode"

	"github.com/PuerkitoBio/goquery"
)

// LessonScraper gets data about lessons from a packed column.
type LessonScraper struct {
	Row    *goquery.Selection
	Lesson *Lesson
}

// LoadLesson scrapes the row and returns a structured lesson.
func (scraper *LessonScraper) LoadLesson() {
	scraper.Lesson = &Lesson{}

	scraper.loadMedia()
	scraper.loadMetadata()
}

func (scraper *LessonScraper) loadMedia() {
	mediaParent := scraper.Row.ChildrenFiltered("td:nth-child(2)")
	title := getSourceTitle(mediaParent)
	source := mediaParent.Find("a").att
}

func (scraper *LessonScraper) loadMetadata() {
	mediaParent := scraper.Row.ChildrenFiltered("td:nth-child(3)")
}

// Gets title as it's specified in the media source column.
func getSourceTitle(dom *goquery.Selection) (title string) {
	title = dom.Text()
	title = strings.TrimFunc(title, func(r rune) bool {
		return unicode.IsSpace(r) || r == rune('-')
	})
	return
}

func getSource(dom *goquery.Selection) (source string) {

}
