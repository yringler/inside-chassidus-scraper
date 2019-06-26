package main

// SiteSection describes a section of a site.
type SiteSection struct {
	Title       string
	Description string
	Sections    []SiteSection
	Lessons     []Lesson
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
