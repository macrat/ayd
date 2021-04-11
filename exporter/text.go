package exporter

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/macrat/ayd/store"
)

func TextExporter(s *store.Store) http.HandlerFunc {
	showIncidentBox := func(w http.ResponseWriter, i *store.Incident, bold bool) {
		banner := "━UNKNOWN━"
		if i.Status == store.STATUS_FAIL {
			banner = "!FAILURE!"
		}

		vert := ""
		if bold {
			fmt.Fprintf(w, "┳━ %s ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┳\n", banner)
			vert = "┃"
		} else {
			fmt.Fprintf(w, "┌─ %s ──────────────────────────────────────────────────────────────────┐\n", banner)
			vert = "│"
		}

		fmt.Fprintf(w, "%s%-78s%s\n", vert, i.Target, vert)

		period := fmt.Sprintf("%s - continue", i.CausedAt.Format(time.RFC3339))
		if !i.ResolvedAt.IsZero() {
			period = fmt.Sprintf("%s - %s", i.CausedAt.Format(time.RFC3339), i.ResolvedAt.Format(time.RFC3339))
		}
		fmt.Fprintf(w, "%s %-77s%s\n", vert, period, vert)

		fmt.Fprint(w, vert, strings.Repeat(" ", 78), vert, "\n")

		for offset := 0; offset < len(i.Message); offset += 78 {
			end := offset + 78
			if end >= len(i.Message) {
				end = len(i.Message) - 1
			}
			fmt.Fprintf(w, "%s%-78s%s\n", vert, i.Message[offset:end], vert)
		}

		if bold {
			fmt.Fprintln(w, "┻━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┻")
		} else {
			fmt.Fprintln(w, "└──────────────────────────────────────────────────────────────────────────────┘")
		}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=UTF-8")

		fmt.Fprintln(w, "───────────────────────────────┤ Current Status ├───────────────────────────────")
		fmt.Fprintln(w)

		for _, history := range s.ProbeHistory.AsSortedArray() {
			fmt.Fprint(w, " ┌─ ", history.Target, "\n")
			fmt.Fprint(w, " └", strings.Repeat("─", store.PROBE_HISTORY_LEN-len(history.Results)))

			for _, r := range history.Results {
				switch r.Status {
				case store.STATUS_OK:
					fmt.Fprintf(w, "✓")
				case store.STATUS_FAIL:
					fmt.Fprintf(w, "!")
				default:
					fmt.Fprintf(w, "━")
				}
			}

			fmt.Fprint(w, "┤  updated: ", history.Results[len(history.Results)-1].CheckedAt.Format(time.RFC3339))

			fmt.Fprintln(w)
		}

		fmt.Fprintln(w)
		fmt.Fprintln(w)
		fmt.Fprintln(w, "──────────────────────────────┤ Incident History ├──────────────────────────────")
		fmt.Fprintln(w)

		for i := range s.CurrentIncidents {
			showIncidentBox(w, s.CurrentIncidents[len(s.CurrentIncidents)-1-i], true)
		}

		for i := range s.IncidentHistory {
			showIncidentBox(w, s.IncidentHistory[len(s.IncidentHistory)-1-i], false)
		}

		fmt.Fprintln(w)
		fmt.Fprintln(w)

		footer := "Reported by Ayd? (" + time.Now().Format(time.RFC3339) + ")"
		pad := strings.Repeat(" ", (80-len(footer))/2)
		fmt.Fprint(w, pad, strings.Repeat("─", len(footer)), "\n")
		fmt.Fprint(w, pad, footer, "\n")
	}
}
