package insidescraper

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"unicode"

	"github.com/PuerkitoBio/goquery"
)

// LessonScraper gets data about lessons from a column.
type LessonScraper struct {
	Row    *goquery.Selection
	Lesson *Lesson
}

// LoadLesson scrapes the row and returns a structured lesson.
func (scraper *LessonScraper) LoadLesson() {
	scraper.Lesson = &Lesson{
		SiteData: &SiteData{
			Title: scraper.Row.ChildrenFiltered("td:first-child").Text(),
		},
		ID: strconv.Itoa(rand.Int()),
	}
	scraper.loadMediaSources()
	scraper.loadMediaDescription()
}

// Creates the media objects, sets source, and title if available.
func (scraper *LessonScraper) loadMediaSources() {
	mediaParent := scraper.Row.ChildrenFiltered("td:nth-child(2)")
	// The media item which is being formed.
	newMedia := &Media{SiteData: &SiteData{}}

	mediaParent.Contents().Each(func(_ int, s *goquery.Selection) {
		switch goquery.NodeName(s) {
		case "br":
			// Line break marks the end of a media item.
			// Reset newMedia for the next media
			newMedia = &Media{SiteData: &SiteData{}}
		case "a":
			if mp3Source, exists := s.Attr("mp3"); exists {
				newMedia.Source = mp3Source
				scraper.Lesson.Audio = append(scraper.Lesson.Audio, *newMedia)
				newMedia = &scraper.Lesson.Audio[len(scraper.Lesson.Audio)-1]
			} else if pdfSource, exists := s.Attr("href"); exists {
				scraper.Lesson.Pdf = append(scraper.Lesson.Pdf, pdfSource)
			} else {
				fmt.Println("Error: No source was found.")
			}
		case "#text":
			newMedia.Title = getSanatizedTitle(s.Text())
		}
	})
}

// Loads description of lesson, and of media, if available.
func (scraper *LessonScraper) loadMediaDescription() {
	mediaParent := scraper.Row.ChildrenFiltered("td:nth-child(3)")
	rawDescription := mediaParent.Text()
	descriptionParts := strings.Split(rawDescription, "\n")

	var activeAudio *Media

	for _, part := range descriptionParts {
		// If the current part is a title, the next parts set  the description of the matching audio.
		// If no matching audio has been found yet, they set the lesson description.

		possibleTitle := getSanatizedTitle(part)
		if matchingAudio := getMediaWithTitle(scraper.Lesson.Audio, possibleTitle); matchingAudio != nil {
			activeAudio = matchingAudio
		} else if activeAudio != nil {
			activeAudio.Description += part + "\n"
		} else {
			scraper.Lesson.Description += part + "\n"
		}
	}

	for i, audio := range scraper.Lesson.Audio {
		scraper.Lesson.Audio[i].Description = strings.TrimFunc(audio.Description, unicode.IsSpace)
	}
	scraper.Lesson.Title = strings.Trim(scraper.Lesson.Title, " \n")
}

// Get title, without the extra bits.
func getSanatizedTitle(title string) string {
	title = strings.Replace(title, "PDF", "", 1)
	title = strings.Replace(title, "MP3", "", 1)

	title = strings.TrimFunc(title, func(r rune) bool {
		return unicode.IsSpace(r) || r == '-' || r == '.' || r == ':'
	})

	return title
}

// Returns pointer to media in the slice with the given title, or nil.
func getMediaWithTitle(mediaSlice []Media, possibleTitle string) *Media {
	for i, value := range mediaSlice {
		if value.Title == possibleTitle {
			return &mediaSlice[i]
		}

		// For cases where the description title has two parts, e.g Class One/ Five (המשך).
		// See https://insidechassidus.org/maamarim/maamarim-of-the-rebbe/text-based-concise-summary/1553-maamarim-5715

		splitPossibleTitle := getSplit(possibleTitle)
		splitActualTitle := getSplit(value.Title)

		// If both titles are composites, then they would already be equal above.
		if len(splitPossibleTitle) < 2 && len(splitActualTitle) < 2 {
			return nil
		}

		// Get the composite and normal titles.
		split := splitPossibleTitle
		normalTitle := value.Title
		if len(split) == 1 {
			split = splitActualTitle
			normalTitle = possibleTitle
		}

		if len(split) == 2 && (split[0] == normalTitle || split[1] == normalTitle) {
			// Use the longer title.
			if len(possibleTitle) > len(mediaSlice[i].Title) {
				mediaSlice[i].Title = possibleTitle
			}
			return &mediaSlice[i]
		}
	}

	return nil
}

func getSplit(title string) []string {
	split := strings.Split(title, "/")
	if len(split) != 2 {
		split = strings.Split(title, ",")
	}

	if len(split) == 2 {
		split[0] = getSanatizedTitle(split[0])
		split[1] = getSanatizedTitle(split[1])
	}

	return split
}
