package endpoint_test

import (
	"testing"
)

func TestIncidentsHTMLEndpoint(t *testing.T) {
	AssertEndpoint(t, "/incidents.html", "./testdata/incidents.html", `Reported by Ayd \(.+\)`)
}

func TestIncidentsRSSEndpoint(t *testing.T) {
	AssertEndpoint(t, "/incidents.rss", "./testdata/incidents.rss", `<pubDate>.+</pubDate>`)
}

func TestIncidentsCSVEndpoint(t *testing.T) {
	AssertEndpoint(t, "/incidents.csv", "./testdata/incidents.csv", "")
}

func TestIncidentsJsonEndpoint(t *testing.T) {
	AssertEndpoint(t, "/incidents.json", "./testdata/incidents.json", `"reported_at":".+?"`)
}
