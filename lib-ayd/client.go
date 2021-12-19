package ayd

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/macrat/ayd/internal/ayderr"
)

// Fetch is fetch Ayd json API and returns Report
func Fetch(u *url.URL) (Report, error) {
	var err error
	u, err = u.Parse("status.json")
	if err != nil {
		return Report{}, ayderr.New(ErrCommunicate, err, "failed to parse URL")
	}

	resp, err := http.Get(u.String())
	if err != nil {
		return Report{}, ayderr.New(ErrCommunicate, err, "failed to fetch")
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)

	var r Report
	if err = dec.Decode(&r); err != nil {
		return Report{}, ayderr.New(ErrCommunicate, err, "failed to parse response")
	}

	return r, nil
}
