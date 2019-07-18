package insidescraper

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
	Pdf   []Media
}

// Media contains information about a particular piece of media.
type Media struct {
	*SiteData
	Source string
}

// SiteData is a base type used by other site structures.
type SiteData struct {
	Title       string
	Description string
}
