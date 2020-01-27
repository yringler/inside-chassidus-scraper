package insidescraper

import (
	"encoding/json"
	"os"
	"testing"
)

func TestResolveSite(t *testing.T) {
	resolver := SectionResolver{
		Site: getSite("data.json"),
	}

	resolver.ResolveSite()

	file, _ := os.Create("data.resolved.json")
	defer file.Close()

	jsonOut, _ := json.MarshalIndent(resolver.ResolvedSite, "", "    ")
	file.Write(jsonOut)
}
