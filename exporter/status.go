package exporter

import (
	_ "embed"
	"encoding/json"
	"fmt"
	htmlTemplate "html/template"
	"io"
	"net/http"
	"strings"
	textTemplate "text/template"

	"github.com/macrat/ayd/store"
)

//go:embed templates/status.html
var statusHTMLTemplate string

func StatusHTMLExporter(s *store.Store) http.HandlerFunc {
	tmpl := htmlTemplate.Must(htmlTemplate.New("status.html").Funcs(templateFuncs).Parse(statusHTMLTemplate))

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")

		HandleError(s, "status.html", tmpl.Execute(w, s.Freeze()))
	}
}

//go:embed templates/status.unicode
var statusUnicodeTextTemplate string

//go:embed templates/status.ascii
var statusASCIITextTemplate string

func StatusTextExporter(s *store.Store) http.HandlerFunc {
	unicode := textTemplate.Must(textTemplate.New("status.unicode").Funcs(templateFuncs).Parse(statusUnicodeTextTemplate))
	ascii := textTemplate.Must(textTemplate.New("status.ascii").Funcs(templateFuncs).Parse(statusASCIITextTemplate))

	return func(w http.ResponseWriter, r *http.Request) {
		var execute func(io.Writer, interface{}) error

		charset := r.URL.Query().Get("charset")
		switch strings.ToLower(charset) {
		case "", "unicode", "utf", "utf8":
			charset = "unicode"
			execute = unicode.Execute
		case "ascii", "us-ascii", "usascii":
			charset = "ascii"
			execute = ascii.Execute
		default:
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusBadRequest)
			_, err := fmt.Fprintln(w, "error: unsupported charset:", charset)
			HandleError(s, "status.txt", err)
			return
		}

		contentType := "text/plain; charset=" + charset
		w.Header().Set("Content-Type", contentType)

		HandleError(s, "status.txt:"+charset, execute(w, s.Freeze()))
	}
}

func StatusJSONExporter(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET")

		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")

		HandleError(s, "status.json", enc.Encode(s.Freeze()))
	}
}
