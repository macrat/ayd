package logconv

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"time"

	api "github.com/macrat/ayd/lib-ayd"
	"github.com/xuri/excelize/v2"
)

func excelPos(x, y uint) string {
	pos, err := excelize.CoordinatesToCellName(int(x+1), int(y+1))
	if err != nil {
		panic(err)
	}
	return pos
}

func contains[T comparable](xs []T, x T) bool {
	for _, a := range xs {
		if x == a {
			return true
		}
	}
	return false
}

func ToXlsx(w io.Writer, s api.LogScanner, createdAt time.Time) error {
	xlsx := excelize.NewFile()
	defer xlsx.Close()
	xlsx.SetSheetName("Sheet1", "log")

	xlsx.SetAppProps(&excelize.AppProperties{
		Application: "Ayd",
	})
	xlsx.SetDocProps(&excelize.DocProperties{
		Created:        createdAt.Format(time.RFC3339),
		Modified:       createdAt.Format(time.RFC3339),
		Creator:        "Ayd",
		LastModifiedBy: "Ayd",
	})

	zone, _ := createdAt.Zone()
	xlsx.SetCellStr("log", "A1", fmt.Sprintf("time (%s)", zone))
	xlsx.SetCellStr("log", "B1", "status")
	xlsx.SetCellStr("log", "C1", "latency")
	xlsx.SetCellStr("log", "D1", "target")
	xlsx.SetCellStr("log", "E1", "message")

	colors := map[api.Status]string{
		api.StatusHealthy: "89C923",
		api.StatusDegrade: "DDA100",
		api.StatusFailure: "FF2D00",
		api.StatusUnknown: "000000",
		api.StatusAborted: "C0C0C0",
	}

	setValue := func(x, y uint, value any, color string, style int, format *string) {
		xlsx.SetCellValue("log", excelPos(x, y), value)
		sid, _ := xlsx.NewStyle(&excelize.Style{
			CustomNumFmt: format,
			Border:       []excelize.Border{{Type: "bottom", Style: style, Color: color}},
		})
		pos := excelPos(x, y)
		xlsx.SetCellStyle("log", pos, pos, sid)
	}
	datefmt := "yyyy-mm-dd hh:mm:ss"
	latencyfmt := "#,##0.000 \"ms\""

	var extras []map[string]interface{}
	var extraKeys []string

	var row uint
	for s.Scan() {
		row++
		if row > 100000 {
			break
		}

		r := s.Record()

		color := colors[r.Status]
		style, _ := xlsx.NewStyle(&excelize.Style{Border: []excelize.Border{{Type: "bottom", Style: 1, Color: color}}})
		xlsx.SetRowStyle("log", int(row+1), int(row+1), style)

		setValue(0, row, r.Time.In(createdAt.Location()), color, 1, &datefmt)
		setValue(1, row, r.Status.String(), color, 5, nil)
		setValue(2, row, float64(r.Latency.Microseconds())/1000, color, 1, &latencyfmt)
		setValue(3, row, r.Target.String(), color, 1, nil)
		setValue(4, row, r.Message, color, 1, nil)

		extras = append(extras, r.Extra)
		for k := range r.Extra {
			if !contains(extraKeys, k) {
				extraKeys = append(extraKeys, k)
			}
		}
	}

	sort.Strings(extraKeys)

	for col, k := range extraKeys {
		pos := excelPos(uint(5+col), 0)
		xlsx.SetCellStr("log", pos, k)
	}

	for row, extra := range extras {
		for col, k := range extraKeys {
			if raw, ok := extra[k]; ok {
				pos := excelPos(uint(5+col), uint(1+row))
				switch v := raw.(type) {
				case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, string, []byte:
					xlsx.SetCellValue("log", pos, v)
				case float32:
					xlsx.SetCellFloat("log", pos, float64(v), 3, 32)
				case float64:
					xlsx.SetCellFloat("log", pos, v, 3, 64)
				default:
					b, err := json.Marshal(v)
					if err == nil {
						xlsx.SetCellStr("log", pos, string(b))
					}
				}
			}
		}
	}

	if err := xlsx.SetPanes("log", `{"freeze":true, "split":false, "x_split":0, "y_split":1, "top_left_cell":"A2", "active_pane":"topLeft"}`); err != nil {
		return err
	}

	xlsx.SetColWidth("log", "A", "A", 20)
	xlsx.SetColWidth("log", "C", "C", 15)
	xlsx.SetColWidth("log", "D", "D", 30)
	xlsx.SetColWidth("log", "E", "E", 30)

	xlsx.AutoFilter("log", "A1", excelPos(uint(4+len(extraKeys)), 0), "")

	return xlsx.Write(w)
}
