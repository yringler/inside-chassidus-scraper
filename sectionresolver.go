package insidescraper

// ResolvedSection stores optimized data.
type ResolvedSection struct {
	*SiteData

	ID string

	Content []ContentReference

	Audio map[string]Media

	// AudioCount contains the total number of audio classes contained in this section,
	// including all descendant sections.
	AudioCount int
}

// ContentReference can refer to any of section, lesson, or media
type ContentReference struct {
	Type DataType

	// Reference can either be the ID of a section or lesson, or a media source URL.
	Reference string
}

// ResolvingItem is an interim object during resolution.
type ResolvingItem struct {
	Type DataType

	SectionID string

	// Audio is for when a section consists only of a single audio.
	Audio *Media

	// Lesson is for when a section is really just a lesson.
	Lesson *Lesson
}

// SectionResolver optimizes data structure.
type SectionResolver struct {
	// Site is the original Site
	Site Site

	// NeweSite is the resolved Site
	ResolvedSite ResolvedSite
}

// ResolvedSite contains all site data.
type ResolvedSite struct {
	Sections map[string]ResolvedSection
	Lessons  map[string]Lesson
	// IDs of all top level sections.
	TopLevel []TopItem
}

// ResolveSite resolves the Site into an optimized site.
func (resolver *SectionResolver) ResolveSite() {
	resolver.ResolvedSite = ResolvedSite{
		TopLevel: resolver.Site.TopLevel,
		Sections: make(map[string]ResolvedSection),
		Lessons:  make(map[string]Lesson),
	}

	for _, topSection := range resolver.Site.TopLevel {
		resolver.ResolveSection(topSection.ID)
	}
}

// ResolveSection converts the given section into its most efficiant
// representation.
func (resolver *SectionResolver) ResolveSection(sectionID string) *ResolvingItem {
	section := resolver.Site.Sections[sectionID]

	if section.AudioCount == 1 {
		if len(section.Lessons) > 0 {
			lesson := resolver.Site.Lessons[section.Lessons[0]]
			media := resolver.ResolveMedia(lesson.Audio[0], lesson)
			return &ResolvingItem{
				Type:  MediaType,
				Audio: &media,
			}
		}
	}

	if !(len(section.Sections) > 0 && len(section.Lessons) > 0) {
		if len(section.Sections) == 1 {
			return resolver.ResolveSection(section.Sections[0])
		}

		if len(section.Sections) > 0 {
			if resolver.isEverySectionMedia(sectionID) {
				return &ResolvingItem{
					Type:   LessonType,
					Lesson: resolver.simpleSectionsToLesson(sectionID),
				}
			}
		} else if resolver.isEveryLessonMedia(sectionID) {
			return &ResolvingItem{
				Type:   LessonType,
				Lesson: resolver.simpleLessonsToLesson(sectionID),
			}
		}
	}

	resolver.ResolvedSite.Sections[sectionID] = ResolvedSection{
		SiteData:   section.SiteData,
		ID:         sectionID,
		AudioCount: section.AudioCount,
	}

	// Finally, if this is a real, complicated section, resolve all of its sub sections.
	for _, subsectionID := range section.Sections {
		resolver.useResolvedToParent(resolver.ResolveSection(subsectionID), sectionID)
	}

	return &ResolvingItem{
		Type:      SectionType,
		SectionID: sectionID,
	}
}

// useResolvedToParent integrates the resolved section into the parent.
func (resolver *SectionResolver) useResolvedToParent(resolved *ResolvingItem, parentID string) {
	parent := resolver.ResolvedSite.Sections[parentID]

	// For a section or lesson, all we have to do is add the reference.
	parent.Content = append(parent.Content)

	// For a lesson, also add it to the lesson map.
	if resolved.Type == MediaType {
		resolver.ResolvedSite.Lessons[resolved.Lesson.ID] = *resolved.Lesson
	}

	resolver.ResolvedSite.Sections[parentID] = parent
}

// simpleContentToLesson converts the given section to a lesson.
func (resolver *SectionResolver) simpleContentToLesson(sectionID string, sourceIDs []string, toAudio func(ID string) Media) *Lesson {
	section := resolver.Site.Sections[sectionID]
	audio := make([]Media, 0)

	for _, ID := range sourceIDs {
		audio = append(audio, toAudio(ID))
	}

	return &Lesson{
		ID:       section.ID,
		SiteData: section.SiteData,
		Audio:    audio,
	}
}

// simpleLessonsToLesson converts from section which has all lessons with one
// class to just one lesson.
func (resolver *SectionResolver) simpleLessonsToLesson(sectionID string) *Lesson {
	return resolver.simpleContentToLesson(sectionID, resolver.Site.Sections[sectionID].Lessons, func(id string) Media {
		return resolver.Site.Lessons[id].Audio[0]
	})
}

// simpleSectionsToLesson converts from all child sections having just one
// lesson to one lesson with all that content.
func (resolver *SectionResolver) simpleSectionsToLesson(sectionID string) *Lesson {
	return resolver.simpleContentToLesson(sectionID, resolver.Site.Sections[sectionID].Lessons, func(id string) Media {
		return *resolver.ResolveSection(id).Audio
	})
}

// IsEverySectionMedia checks if the given section is really
// just a lesson.
func (resolver *SectionResolver) isEverySectionMedia(sectionID string) bool {
	section := resolver.Site.Sections[sectionID]

	for _, subSection := range section.Sections {
		if resolver.Site.Sections[subSection].AudioCount > 1 {
			return false
		}
	}

	return true
}

// isEveryLessonMedia checks if all lessons in section have just one media.
func (resolver *SectionResolver) isEveryLessonMedia(sectionID string) bool {
	section := resolver.Site.Sections[sectionID]

	for _, lessonID := range section.Lessons {
		if len(resolver.Site.Lessons[lessonID].Audio) > 1 {
			return false
		}
	}

	return true
}

// ResolveMedia gives the given media all of its data.
func (resolver *SectionResolver) ResolveMedia(audio Media, lesson Lesson) Media {
	title := audio.Title

	if len(title) == 0 {
		title = lesson.Title
	}

	description := audio.Description

	if len(description) == 0 {
		description = lesson.Description
	}

	audio.Title = title
	audio.Description = description
	return audio
}
