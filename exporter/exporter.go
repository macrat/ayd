package exporter

import (
	_ "embed"
	"net/http"
	"net/url"
	"time"

	"github.com/NYTimes/gziphandler"
	api "github.com/macrat/ayd/lib-ayd"
	"github.com/macrat/ayd/store"
)

//go:embed static/favicon.ico
var faviconIco []byte

//go:embed static/favicon.svg
var faviconSvg []byte

//go:embed static/not-found.html
var notFoundPage []byte

func New(s *store.Store) http.Handler {
	m := http.NewServeMux()

	m.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Write(faviconIco)
	})
	m.HandleFunc("/favicon.svg", func(w http.ResponseWriter, r *http.Request) {
		w.Write(faviconSvg)
	})

	m.Handle("/status", http.RedirectHandler("/status.html", http.StatusMovedPermanently))
	m.HandleFunc("/status.txt", StatusTextExporter(s))
	m.HandleFunc("/status.html", StatusHTMLExporter(s))
	m.HandleFunc("/status.json", StatusJSONExporter(s))

	m.Handle("/log", http.RedirectHandler("/log.tsv", http.StatusMovedPermanently))
	m.HandleFunc("/log.tsv", LogTSVExporter(s))

	m.HandleFunc("/metrics", MetricsExporter(s))
	m.HandleFunc("/healthz", HealthzExporter(s))

	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/status.html", http.StatusFound)
		} else {
			w.WriteHeader(http.StatusNotFound)
			w.Write(notFoundPage)
		}
	})

	return gziphandler.GzipHandler(m)
}

func HandleError(s *store.Store, scope string, err error) {
	if err != nil {
		s.Report(api.Record{
			CheckedAt: time.Now(),
			Target:    &url.URL{Scheme: "ayd", Opaque: "api:" + scope},
			Status:    api.StatusFailure,
			Message:   err.Error(),
		})
	}
}
