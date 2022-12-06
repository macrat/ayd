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

func ToXlsx(w io.Writer, s api.LogScanner) error {
	var extras []map[string]interface{}
	var extraKeys []string

	xlsx := excelize.NewFile()
	defer xlsx.Close()
	xlsx.SetSheetName("Sheet1", "log")

	xlsx.SetAppProps(&excelize.AppProperties{
		Application: "Ayd",
	})
	timestamp := time.Now().Format(time.RFC3339)
	xlsx.SetDocProps(&excelize.DocProperties{
		Created:        timestamp,
		Modified:       timestamp,
		Creator:        "Ayd",
		LastModifiedBy: "Ayd",
	})

	xlsx.SetCellStr("log", "A1", "time")
	xlsx.SetCellStr("log", "B1", "status")
	xlsx.SetCellStr("log", "C1", "latency")
	xlsx.SetCellStr("log", "D1", "target")
	xlsx.SetCellStr("log", "E1", "message")

	var row uint
	for s.Scan() {
		row++
		r := s.Record()

		xlsx.SetCellValue("log", excelPos(0, row), r.Time.UTC())
		xlsx.SetCellStr("log", excelPos(1, row), r.Status.String())
		xlsx.SetCellFloat("log", excelPos(2, row), float64(r.Latency.Microseconds())/1000, 3, 64)
		xlsx.SetCellStr("log", excelPos(3, row), r.Target.String())
		xlsx.SetCellStr("log", excelPos(4, row), r.Message)

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

	datefmt := "yyyy-mm-dd hh:mm:ss\"Z\""
	if style, err := xlsx.NewStyle(&excelize.Style{CustomNumFmt: &datefmt}); err == nil {
		xlsx.SetColStyle("log", "A", style)
	}
	xlsx.SetColWidth("log", "A", "A", 20)

	healthy, _ := xlsx.NewConditionalStyle(`{"border": [{"type":"bottom", "style":5, "color":"89C923"}]}`)
	degrade, _ := xlsx.NewConditionalStyle(`{"border": [{"type":"bottom", "style":5, "color":"DDA100"}]}`)
	failure, _ := xlsx.NewConditionalStyle(`{"border": [{"type":"bottom", "style":5, "color":"FF2D00"}]}`)
	unknown, _ := xlsx.NewConditionalStyle(`{"border": [{"type":"bottom", "style":5, "color":"000000"}]}`)
	xlsx.SetConditionalFormat("log", "B:B", fmt.Sprintf(`[
		{"type":"cell", "criteria":"==", "value":"\"HEALTHY\"", "format":%d},
		{"type":"cell", "criteria":"==", "value":"\"DEGRADE\"", "format":%d},
		{"type":"cell", "criteria":"==", "value":"\"FAILURE\"", "format":%d},
		{"type":"cell", "criteria":"==", "value":"\"UNKNOWN\"", "format":%d}
	]`, healthy, degrade, failure, unknown))

	healthy, _ = xlsx.NewConditionalStyle(`{"border": [{"type":"bottom", "style":1, "color":"89C923"}]}`)
	degrade, _ = xlsx.NewConditionalStyle(`{"border": [{"type":"bottom", "style":2, "color":"DDA100"}]}`)
	failure, _ = xlsx.NewConditionalStyle(`{"border": [{"type":"bottom", "style":2, "color":"FF2D00"}]}`)
	unknown, _ = xlsx.NewConditionalStyle(`{"border": [{"type":"bottom", "style":2, "color":"000000"}]}`)
	aborted, _ := xlsx.NewConditionalStyle(`{"border": [{"type":"bottom", "style":1, "color":"000000"}]}`)
	endCol, err := excelize.ColumnNumberToName(5 + len(extraKeys))
	if err != nil {
		endCol = "ZZ"
	}
	xlsx.SetConditionalFormat("log", "A:"+endCol, fmt.Sprintf(`[
		{"type":"formula", "criteria":"=$B1=\"HEALTHY\"", "format":%d},
		{"type":"formula", "criteria":"=$B1=\"DEGRADE\"", "format":%d},
		{"type":"formula", "criteria":"=$B1=\"FAILURE\"", "format":%d},
		{"type":"formula", "criteria":"=$B1=\"UNKNOWN\"", "format":%d},
		{"type":"formula", "criteria":"=$B1=\"ABORTED\"", "format":%d}
	]`, healthy, degrade, failure, unknown, aborted))

	latencyfmt := "#,##0.000 \"ms\""
	if style, err := xlsx.NewStyle(&excelize.Style{CustomNumFmt: &latencyfmt}); err == nil {
		xlsx.SetColStyle("log", "C", style)
	}
	xlsx.SetColWidth("log", "C", "C", 15)

	xlsx.SetColWidth("log", "D", "D", 30)

	xlsx.AutoFilter("log", "A1", excelPos(uint(4+len(extraKeys)), 0), "")

	return xlsx.Write(w)
}
