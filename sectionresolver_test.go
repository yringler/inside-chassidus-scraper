package insidescraper

import (
	"encoding/json"
	"os"
	"testing"
)

func TestResolveSite(t *testing.T) {
	runResolver(getSite("data.json"), "data.resolved.json")
}

func runResolver(site Site, resolvedName string) {
	resolver := SectionResolver{
		Site: getSite("data.json"),
	}

	resolver.ResolveSite()

	file, _ := os.Create(resolvedName)
	defer file.Close()

	jsonOut, _ := json.MarshalIndent(resolver.ResolvedSite, "", "    ")
	file.Write(jsonOut)
}
