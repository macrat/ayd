package endpoint

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	textTemplate "text/template"
)

//go:embed templates/status.html
var statusHTMLTemplate string

func StatusHTMLEndpoint(s Store) http.HandlerFunc {
	tmpl := loadHTMLTemplate(statusHTMLTemplate)

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")

		handleError(s, "status.html", tmpl.Execute(w, s.MakeReport(20)))
	}
}

//go:embed templates/status.unicode
var statusUnicodeTextTemplate string

//go:embed templates/status.ascii
var statusASCIITextTemplate string

func StatusTextEndpoint(s Store) http.HandlerFunc {
	unicode := textTemplate.Must(textTemplate.New("status.unicode").Funcs(templateFuncs).Parse(statusUnicodeTextTemplate))
	ascii := textTemplate.Must(textTemplate.New("status.ascii").Funcs(templateFuncs).Parse(statusASCIITextTemplate))

	return func(w http.ResponseWriter, r *http.Request) {
		var execute func(io.Writer, interface{}) error

		charset := r.URL.Query().Get("charset")
		switch strings.ToLower(charset) {
		case "", "unicode", "utf", "utf8":
			charset = "UTF-8"
			execute = unicode.Execute
		case "ascii", "us-ascii", "usascii":
			charset = "ascii"
			execute = ascii.Execute
		default:
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusBadRequest)
			_, err := fmt.Fprintln(w, "error: given unsupported charset. please use \"utf-8\", \"ascii\", or remove charset query.")
			handleError(s, "status.txt", err)
			return
		}

		contentType := "text/plain; charset=" + charset
		w.Header().Set("Content-Type", contentType)

		handleError(s, "status.txt:"+charset, execute(w, s.MakeReport(40)))
	}
}

func StatusJSONEndpoint(s Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET")

		enc := json.NewEncoder(w)

		handleError(s, "status.json", enc.Encode(s.MakeReport(40)))
	}
}
