package insidescraper

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestCount(t *testing.T) {
	site := getSite("data.json")
	counter := LessonCounter{
		Data: &site,
	}
	counter.CountLessons()
	output, _ := json.MarshalIndent(site, "", "    ")
	fmt.Println(string(output))
}
