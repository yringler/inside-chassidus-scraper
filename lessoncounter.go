package insidescraper

// LessonCounter sets the lesson count property of each section.
// This is kept seperate from the scraper so that postfixes etc can be
// applied to the data before counting everything up.
type LessonCounter struct {
	Data *Site
}

// CountLessons counts all the lessons, recursively.
func (counter *LessonCounter) CountLessons() {
	for _, topItem := range counter.Data.TopLevel {
		counter.countLessons(topItem.ID)
	}
}

func (counter *LessonCounter) countLessons(sectionID string) int {
	section := counter.Data.Sections[sectionID]
	if section.isBeingCounted {
		return 0
	}
	if section.AudioCount > 0 {
		return section.AudioCount
	}
	section.isBeingCounted = true
	counter.Data.Sections[sectionID] = section

	for _, id := range section.Sections {
		section.AudioCount += counter.countLessons(id)
	}

	for _, id := range section.Lessons {
		section.AudioCount += len(counter.Data.Lessons[id].Audio)
	}

	section.isBeingCounted = false
	counter.Data.Sections[sectionID] = section

	return section.AudioCount
}
