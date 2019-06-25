package main

import (
	"github.com/gocolly/colly"
	"fmt"
)

func main() {
	c := colly.NewCollector()

	c.OnError(func(_ *colly.Response, err error) {
		fmt.Println("Something went wrong:", err)
	})

	c.OnHTML("body.home a.dropdown-toggle", func (e *colly.HTMLElement)  {
		fmt.Println(e.Text)
	})

	c.OnResponse(func(r *colly.Response) {
		fmt.Println("Visited", r.Request.URL)
	})

	c.Visit("http://insidechassidus.org/")
}