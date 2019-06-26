package main

// SiteSection describes a section of a site.
type SiteSection struct {
	Title       string
	Description string
	Sections    []SiteSection
	Lessons     []Lesson
}

// Lesson describes one particular audio class.
type Lesson struct {
	Title       string
	Description string
	Audio       string
	Pdf         string
}
