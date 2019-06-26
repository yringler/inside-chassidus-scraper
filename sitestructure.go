package main

// SiteSection describes a section of a site.
type SiteSection struct {
	Title       string
	Description string
	Sections    []SiteSection
	Lessons     []Lesson
}

// Lesson describes one lesson. It may contain multiple classes.
type Lesson struct {
	Title       string
	Description string
	Audio       []string
	Pdf         []string
}
