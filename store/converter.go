package store

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/macrat/ayd/probe"
)

func str2result(s string) (probe.Result, error) {
	var r probe.Result
	var timestamp string
	var latency float64
	var target string

	ss := strings.SplitN(s, "\t", 5)
	if len(ss) != 5 {
		return probe.Result{}, fmt.Errorf("unexpected value count")
	}

	timestamp = ss[0]
	r.Status = probe.ParseStatus(ss[1])
	latency, err := strconv.ParseFloat(ss[2], 64)
	if err != nil {
		return probe.Result{}, err
	}
	target = ss[3]
	r.Message = ss[4]

	r.CheckedAt, err = time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return probe.Result{}, err
	}

	r.Latency = time.Duration(latency * float64(time.Millisecond))

	r.Target, err = url.Parse(target)
	if err != nil {
		return probe.Result{}, err
	}

	return r, nil
}

func result2str(r probe.Result, humanReadable bool) string {
	msec := ""
	if humanReadable {
		msec = "msec"
	}

	return strings.Join([]string{
		r.CheckedAt.Format(time.RFC3339),
		r.Status.String(),
		fmt.Sprintf("%.3f%s", float64(r.Latency.Microseconds())/1000, msec),
		r.Target.String(),
		strings.ReplaceAll(strings.ReplaceAll(r.Message, "\t", "    "), "\n", " "),
	}, "\t")
}
