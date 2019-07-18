package insidescraper

// SiteData is a base type used by other site structures.
type SiteData struct {
	Title       string
	Description string
}

// SiteSection describes a section of a site.
type SiteSection struct {
	*SiteData

	ID string
	// Sections contains the ids of all sub sections
	Sections []string
	Lessons  []Lesson
	// Wether this section is a top level section
	IsTopLevel bool
}

// Media contains information about a particular piece of media.
type Media struct {
	*SiteData
	Source string
}

// Lesson describes one lesson. It may contain multiple classes.
type Lesson struct {
	*SiteData
	Audio []Media
	Pdf   []Media
}
