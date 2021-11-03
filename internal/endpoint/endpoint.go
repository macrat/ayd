package endpoint

import (
	_ "embed"
	"net/http"

	"github.com/NYTimes/gziphandler"
)

//go:embed static/favicon.ico
var faviconIco []byte

//go:embed static/favicon.svg
var faviconSvg []byte

//go:embed static/not-found.html
var notFoundPage []byte

func New(s Store) http.Handler {
	m := http.NewServeMux()

	m.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Write(faviconIco)
	})
	m.HandleFunc("/favicon.svg", func(w http.ResponseWriter, r *http.Request) {
		w.Write(faviconSvg)
	})

	m.Handle("/status", http.RedirectHandler("/status.html", http.StatusMovedPermanently))
	m.HandleFunc("/status.txt", StatusTextEndpoint(s))
	m.HandleFunc("/status.html", StatusHTMLEndpoint(s))
	m.HandleFunc("/status.json", StatusJSONEndpoint(s))

	m.Handle("/log", http.RedirectHandler("/log.tsv", http.StatusMovedPermanently))
	m.HandleFunc("/log.tsv", LogTSVEndpoint(s))
	m.HandleFunc("/log.json", LogJsonEndpoint(s))
	m.HandleFunc("/log.csv", LogCSVEndpoint(s))

	m.Handle("/targets", http.RedirectHandler("/targets.txt", http.StatusMovedPermanently))
	m.HandleFunc("/targets.txt", TargetsTextEndpoint(s))
	m.HandleFunc("/targets.json", TargetsJSONEndpoint(s))

	m.HandleFunc("/metrics", MetricsEndpoint(s))
	m.HandleFunc("/healthz", HealthzEndpoint(s))

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

func HandleError(s Store, scope string, err error) {
	if err != nil {
		s.ReportInternalError("endpoint:"+scope, err.Error())
	}
}
