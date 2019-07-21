package insidescraper

import (
	"errors"
)

// Site contains all site data.
type Site struct {
	Sections map[string]SiteSection
	Lessons  map[string]Lesson
	// IDs of all top level sections.
	TopLevel []string
}

// SiteSection describes a section of a site.
type SiteSection struct {
	*SiteData

	ID string
	// Sections contains the ids of all sub sections
	Sections []string
	// IDs of all lessons in this section.
	Lessons []string
}

// Lesson describes one lesson. It may contain multiple classes.
type Lesson struct {
	*SiteData
	// ID is the URL of the lessons, if they are from their own page.
	ID    string
	Audio []Media
}

// Media contains information about a particular piece of media.
type Media struct {
	// Note that a media item will *only* have it's own PDF if it was converted from a lesson.
	*SiteData
	Source string
}

// SiteData is a base type used by other site structures.
type SiteData struct {
	Title       string
	Description string
	// PDFs can pop up at any level.
	// For example, sometimes a section has a pdf for it. This usually (proably always) happens
	// when the section contains only lessons, when it will anyway be converted to a lesson.
	// See https://insidechassidus.org/thought-and-history/123-kabbala-and-philosophy-series/1699-chassidus-understanding-what-can-be-understood-of-g-dliness/section-one-before-logic
	Pdf string
}

// ConvertToLesson converts the section to a lesson if it only contains single-audio lessons.
// Returns an error if it can't be done.
func (site *Site) ConvertToLesson(sectionID string) error {
	section := site.Sections[sectionID]

	// Make sure it doesn't contain sections.
	if len(section.Sections) > 0 {
		return errors.New("Contains sections")
	}

	switch len(section.Lessons) {
	case 0:
		return errors.New("Does not contain any lessons")
	case 1:
		// The section is really just a single lesson. Get rid of the pretend section.
		delete(site.Sections, sectionID)

		/*
		 * Change the ID and key of the lesson to the old section ID.
		 * This will allow any references to it to be found and updated.
		 */

		lesson := site.Lessons[section.Lessons[0]]
		oldLessonID := lesson.ID
		lesson.ID = sectionID
		site.Lessons[lesson.ID] = lesson
		// The lesson is no longer locatable with it's old ID.
		delete(site.Lessons, oldLessonID)
		return nil
	default:
		// A section can only be converted to a lesson if all of it's lessons are single audio files.
		for _, lessonID := range section.Lessons {
			if lesson, exists := site.Lessons[lessonID]; exists {
				if len(lesson.Audio) > 1 {
					return errors.New("Contains complex lessons: " + lesson.Title + "," + sectionID)
				}
			} else {
				panic("Hey, why doesn't " + lessonID + " (referenced by " + section.ID + ")" + " exist?")
			}
		}
	}

	// Move section over to lesson.
	site.Lessons[sectionID] = site.getLessonFromSection(sectionID)
	delete(site.Sections, sectionID)

	return nil
}

// Creates one lesson from a section which consists only of single media lessons.
func (site *Site) getLessonFromSection(sectionID string) Lesson {
	section := site.Sections[sectionID]

	// Create lesson.
	newLesson := Lesson{
		SiteData: section.SiteData,
		ID:       section.ID,
		Audio:    make([]Media, 0, len(section.Lessons)),
	}

	// Move the section's lessons into media on this one, new lesson.

	for _, lessonID := range section.Lessons {
		lessonToConvert := site.Lessons[lessonID]

		if len(lessonToConvert.Audio) != 0 {
			newLesson.Audio = append(newLesson.Audio, Media{
				// Note that SiteData also includes PDF URL.
				SiteData: lessonToConvert.SiteData,
				Source:   lessonToConvert.Audio[0].Source,
			})
		}

		// Delete the old, single media lesson.
		delete(site.Lessons, lessonID)
	}

	return newLesson
}
