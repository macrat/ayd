package endpoint

import (
	"bytes"
	_ "embed"
	"net/http"

	"github.com/NYTimes/gziphandler"
)

type CommonHeader struct {
	Upstream http.Handler
}

func (ch CommonHeader) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Server", "ayd")
	ch.Upstream.ServeHTTP(w, r)
}

type LinkHeader struct {
	Upstream http.HandlerFunc
	Link     string
}

func (lh LinkHeader) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Link", lh.Link)
	lh.Upstream.ServeHTTP(w, r)
}

//go:embed static/favicon.ico
var faviconIco []byte

//go:embed static/favicon.svg
var faviconSvg []byte

//go:embed templates/not-found.html
var notFoundPageTemplate string

// New makes new http.Handler
func New(s Store) http.Handler {
	m := http.NewServeMux()

	faviconLink := `<favicon.ico>;rel="alternate";type="image/vnd.microsoft.icon", <favicon.svg>;rel="alternate";type="image/svg+xml"`
	m.Handle("/favicon.ico", LinkHeader{func(w http.ResponseWriter, r *http.Request) {
		w.Write(faviconIco)
	}, faviconLink})
	m.Handle("/favicon.svg", LinkHeader{func(w http.ResponseWriter, r *http.Request) {
		w.Write(faviconSvg)
	}, faviconLink})

	statusLink := `<status.html>;rel="alternate";type="text/html", <status.html>;rel="alternate";type="text/plain", <status.json>;rel="alternate";type="application/json"`
	m.Handle("/status", http.RedirectHandler("/status.html", http.StatusMovedPermanently))
	m.Handle("/status.html", LinkHeader{StatusHTMLEndpoint(s), statusLink})
	m.Handle("/status.txt", LinkHeader{StatusTextEndpoint(s), statusLink})
	m.Handle("/status.json", LinkHeader{StatusJSONEndpoint(s), statusLink})

	incidentsLink := `<incidents.html>;rel="alternate";type="text/html", <incidents.rss>;rel="alternate";type="application/rss+xml", <incidents.csv>;rel="alternate";type="text/csv", <incidents.json>;rel="alternate";type="application/json"`
	m.Handle("/incidents", http.RedirectHandler("/incidents.html", http.StatusMovedPermanently))
	m.Handle("/incidents.html", LinkHeader{IncidentsHTMLEndpoint(s), incidentsLink})
	m.Handle("/incidents.rss", LinkHeader{IncidentsRSSEndpoint(s), incidentsLink})
	m.Handle("/incidents.csv", LinkHeader{IncidentsCSVEndpoint(s), incidentsLink})
	m.Handle("/incidents.json", LinkHeader{IncidentsJSONEndpoint(s), incidentsLink})

	logLink := `<log.html>;rel="alternate";type="text/html", <log.csv>;rel="alternate";type="text/csv", <log.xlsx>;rel="alternate";type="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", <log.ltsv>;rel="alternate";type="text/plain", <log.json>;rel="alternate";type="application/json"`
	m.Handle("/log", http.RedirectHandler("/log.html", http.StatusMovedPermanently))
	m.Handle("/log.html", LinkHeader{LogHTMLEndpoint(s), logLink})
	m.Handle("/log.csv", LinkHeader{LogCSVEndpoint(s), logLink})
	m.Handle("/log.xlsx", LinkHeader{LogXlsxEndpoint(s), logLink})
	m.Handle("/log.ltsv", LinkHeader{LogLTSVEndpoint(s), logLink})
	m.Handle("/log.json", LinkHeader{LogJsonEndpoint(s), logLink})

	targetsLink := `<targets.txt>;rel="alternate";type="text/plain", <targets.json>;rel="alternate";type="application/json"`
	m.Handle("/targets", http.RedirectHandler("/targets.txt", http.StatusMovedPermanently))
	m.Handle("/targets.txt", LinkHeader{TargetsTextEndpoint(s), targetsLink})
	m.Handle("/targets.json", LinkHeader{TargetsJSONEndpoint(s), targetsLink})

	m.HandleFunc("/metrics", MetricsEndpoint(s))
	m.HandleFunc("/healthz", HealthzEndpoint(s))

	buf := bytes.NewBuffer(nil)
	if err := loadHTMLTemplate(notFoundPageTemplate).Execute(buf, nil); err != nil {
		panic(err)
	}
	notFoundPage := buf.Bytes()
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/status.html", http.StatusFound)
		} else {
			w.WriteHeader(http.StatusNotFound)
			w.Write(notFoundPage)
		}
	})

	return gziphandler.GzipHandler(CommonHeader{m})
}

func handleError(s Store, scope string, err error) {
	if err != nil {
		s.ReportInternalError("endpoint:"+scope, err.Error())
	}
}
