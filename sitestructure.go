package insidescraper

// SiteSection describes a section of a site.
type SiteSection struct {
	ID          string
	Title       string
	Description string
	// Sections contains the ids of all sub sections
	Sections []string
	Lessons  []Lesson
	// Wether this section is a top level section
	IsTopLevel bool
}

// Media contains information about a particular piece of media.
type Media struct {
	Title  string
	Source string
}

// Lesson describes one lesson. It may contain multiple classes.
type Lesson struct {
	Title       string
	Description string
	Audio       []string
	Pdf         []Media
}
