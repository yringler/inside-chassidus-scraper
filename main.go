package main

import (
	"fmt"

	"github.com/gocolly/colly"
)

func main() {
	c := colly.NewCollector()

	c.OnError(func(_ *colly.Response, err error) {
		fmt.Println("Something went wrong:", err)
	})

	c.OnHTML("body.home #main-menu-fst > li > a ", func(e *colly.HTMLElement) {
		fmt.Println(e.Attr("href"))
		fmt.Println(e.Text)
	})

	c.Visit("http://insidechassidus.org/")
}
