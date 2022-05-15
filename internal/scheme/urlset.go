package scheme

import (
	"sort"
	"strings"

	api "github.com/macrat/ayd/lib-ayd"
)

// urlSet is a set of URL.
type urlSet []*api.URL

func (s urlSet) search(u *api.URL) int {
	return sort.Search(len(s), func(i int) bool {
		return strings.Compare(s[i].String(), u.String()) <= 0
	})
}

// Has check if the URL is in this urlSet or not.
func (s urlSet) Has(u *api.URL) bool {
	i := s.search(u)
	if len(s) == i {
		return false
	}

	return s[i].String() == u.String()
}

// Add adds a URL to urlSet.
// If the URL is already added, it will be ignored.
func (s *urlSet) Add(u *api.URL) {
	i := s.search(u)
	if len(*s) == i {
		*s = append(*s, u)
	}

	if (*s)[i].String() != u.String() {
		*s = append(append((*s)[:i], u), (*s)[i:]...)
	}
}
