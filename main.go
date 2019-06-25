package main

import (
	"github.com/gocolly/colly"
	"fmt"
)

func main() {
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/70.0.3538.77 Safari/537.36"),
	)

	c.OnError(func(_ *colly.Response, err error) {
		fmt.Println("Something went wrong:", err)
	})

	c.OnHTML("body.home #main-menu-fst > li > a ", func (e *colly.HTMLElement)  {
		fmt.Println(e.Attr("href"))
		fmt.Println(e.Text)
	})

	c.Visit("http://insidechassidus.org/")
}