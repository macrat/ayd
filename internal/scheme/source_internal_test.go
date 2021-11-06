package scheme

import (
	"fmt"
	"net/url"
	"testing"
)

func TestURLSet(t *testing.T) {
	x, _ := url.Parse("https://example.com")
	y, _ := url.Parse("dummy:")
	z, _ := url.Parse("https://example.com")

	s := &urlSet{}

	if s.Has(x) {
		t.Errorf("should not have x yet, but urlSet reported true: %v", s)
	}
	if s.Has(y) {
		t.Errorf("should not have y yet, but urlSet reported true: %v", s)
	}
	if s.Has(z) {
		t.Errorf("should not have z yet, but urlSet reported true: %v", s)
	}
	if len(*s) != 0 {
		t.Errorf("expected length is 0 but got %d", len(*s))
	}

	s.Add(x)
	if !s.Has(x) {
		t.Errorf("should have x, but urlSet reported false: %v", s)
	}
	if s.Has(y) {
		t.Errorf("should not have y yet, but urlSet reported true: %v", s)
	}
	if !s.Has(z) {
		t.Errorf("should have z, but urlSet reported false: %v", s)
	}
	if len(*s) != 1 {
		t.Errorf("expected length is 1 but got %d", len(*s))
	}

	s.Add(y)
	if !s.Has(x) {
		t.Errorf("should have x, but urlSet reported false: %v", s)
	}
	if !s.Has(y) {
		t.Errorf("should have y, but urlSet reported false: %v", s)
	}
	if !s.Has(z) {
		t.Errorf("should have z, but urlSet reported false: %v", s)
	}
	if len(*s) != 2 {
		t.Errorf("expected length is 2 but got %d", len(*s))
	}

	s.Add(z)
	if !s.Has(x) {
		t.Errorf("should have x, but urlSet reported false: %v", s)
	}
	if !s.Has(y) {
		t.Errorf("should have y, but urlSet reported false: %v", s)
	}
	if !s.Has(z) {
		t.Errorf("should have z, but urlSet reported false: %v", s)
	}
	if len(*s) != 2 {
		t.Errorf("expected length is 2 but got %d", len(*s))
	}
}

func BenchmarkURLSet_Add(b *testing.B) {
	us := make([]*url.URL, 1000)

	for i := range us {
		us[i] = &url.URL{Scheme: "dummy", Fragment: fmt.Sprintf("dummy-%d", i)}
	}

	s := &urlSet{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 10000; j++ {
			s.Add(us[j%len(us)])
		}
	}
}
