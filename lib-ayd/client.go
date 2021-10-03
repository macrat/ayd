package ayd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// Fetch is fetch Ayd json API and returns Report
func Fetch(u *url.URL) (Report, error) {
	var err error
	u, err = u.Parse("status.json")
	if err != nil {
		return Report{}, fmt.Errorf("failed to parse URL: %w", err)
	}

	resp, err := http.Get(u.String())
	if err != nil {
		return Report{}, fmt.Errorf("failed to fetch: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return Report{}, fmt.Errorf("failed to read response: %w", err)
	}

	var r Report
	err = json.Unmarshal(raw, &r)
	if err != nil {
		return Report{}, fmt.Errorf("failed to parse response: %w", err)
	}

	return r, nil
}
